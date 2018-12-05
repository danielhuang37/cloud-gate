package httpd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/cloud-gate/broker"
	"github.com/Symantec/cloud-gate/broker/configuration"
	"github.com/Symantec/cloud-gate/broker/staticconfiguration"
	"github.com/Symantec/cloud-gate/lib/constants"
	"github.com/Symantec/cloud-gate/lib/userinfo"
	"github.com/cviecco/go-simple-oidc-auth/authhandler"

	"golang.org/x/net/context"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

type AuthCookie struct {
	Username  string
	ExpiresAt time.Time
}

type Server struct {
	brokers      map[string]broker.Broker
	config       *configuration.Configuration
	htmlWriters  []HtmlWriter
	htmlTemplate *template.Template
	logger       log.DebugLogger
	cookieMutex  sync.Mutex
	authCookie   map[string]AuthCookie
	authSource   *authhandler.SimpleOIDCAuth
	staticConfig *staticconfiguration.StaticConfiguration
	userInfo     userinfo.UserInfo
}

func StartServer(staticConfig *staticconfiguration.StaticConfiguration,
	userInfo userinfo.UserInfo,
	brokers map[string]broker.Broker,
	logger log.DebugLogger) (*Server, error) {

	statusListener, err := net.Listen("tcp", fmt.Sprintf(":%d", staticConfig.Base.StatusPort))
	if err != nil {
		return nil, err
	}
	serviceListener, err := net.Listen("tcp", fmt.Sprintf(":%d", staticConfig.Base.ServicePort))
	if err != nil {
		return nil, err
	}
	server := &Server{
		brokers:      brokers,
		logger:       logger,
		userInfo:     userInfo,
		staticConfig: staticConfig,
	}
	server.authCookie = make(map[string]AuthCookie)
	go server.performStateCleanup(constants.SecondsBetweenCleanup)

	// load templates
	server.htmlTemplate = template.New("main")
	// Load the other built in templates
	extraTemplates := []string{footerTemplateText, consoleAccessTemplateText, generateTokaneTemplateText, headerTemplateText}
	for _, templateString := range extraTemplates {
		_, err = server.htmlTemplate.Parse(templateString)
		if err != nil {
			return nil, err
		}
	}

	http.HandleFunc("/", server.dashboardRootHandler)
	http.HandleFunc("/status", server.statusHandler)
	serviceMux := http.NewServeMux()
	serviceMux.HandleFunc("/", server.consoleAccessHandler)
	serviceMux.HandleFunc("/getconsole", server.getConsoleUrlHandler)
	serviceMux.HandleFunc("/generatetoken", server.generateTokenHandler)
	serviceMux.HandleFunc("/static/", staticHandler)

	//setup openidc auth
	ctx := context.Background()
	simpleOidcAuth := authhandler.NewSimpleOIDCAuth(&ctx, staticConfig.OpenID.ClientID, staticConfig.OpenID.ClientSecret, staticConfig.OpenID.ProviderURL)
	server.authSource = simpleOidcAuth
	serviceMux.Handle(constants.OidcCallbackPath, simpleOidcAuth.Handler(http.HandlerFunc(server.consoleAccessHandler)))
	serviceMux.Handle(constants.LoginPath, simpleOidcAuth.Handler(http.HandlerFunc(server.loginHandler)))

	statusServer := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		err := statusServer.Serve(statusListener)
		if err != nil {
			logger.Fatalf("Failed to start status server, err=%s", err)
		}
	}()

	var clientCACertPool *x509.CertPool
	if len(staticConfig.Base.ClientCAFilename) > 0 {
		clientCACertPool = x509.NewCertPool()
		caCert, err := ioutil.ReadFile(staticConfig.Base.ClientCAFilename)
		if err != nil {
			logger.Fatalf("cannot read clientCA file err=%s", err)
		}
		clientCACertPool.AppendCertsFromPEM(caCert)
	}

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		},
		ClientAuth: tls.VerifyClientCertIfGiven,
		ClientCAs:  clientCACertPool,
	}
	serviceServer := &http.Server{
		Handler:      serviceMux,
		TLSConfig:    tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		err := serviceServer.ServeTLS(serviceListener, staticConfig.Base.TLSCertFilename, staticConfig.Base.TLSKeyFilename)
		if err != nil {
			logger.Fatalf("Failed to start service server, err=%s", err)
		}
	}()
	return server, nil
}

func (s *Server) AddHtmlWriter(htmlWriter HtmlWriter) {
	s.htmlWriters = append(s.htmlWriters, htmlWriter)
}

func (s *Server) UpdateConfiguration(
	config *configuration.Configuration) error {
	s.config = config
	return nil
}