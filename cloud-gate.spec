Name:           cloud-gate
Version:        0.6.2
Release:        1%{?dist}
Summary:        Access broker for clouds

#Group:
License:        ASL 2.0
URL:            https://github.com/cviecco/simple-cloud-encrypt/
Source0:        cloud-gate-%{version}.tar.gz

#BuildRequires: golang
#Requires:
Requires(pre): /usr/sbin/useradd, /usr/bin/getent
Requires(postun): /usr/sbin/userdel

#no debug package as this is go
%define debug_package %{nil}

%description
A web broker for accesing AWS and potentially other clouds


%prep
%setup -n %{name}-%{version}


%build
make


%install
#%make_install
%{__install} -Dp -m0755 ~/go/bin/cloud-gate %{buildroot}%{_bindir}/cloud-gate
install -d %{buildroot}/usr/lib/systemd/system
install -p -m 0644 misc/startup/cloud-gate.service %{buildroot}/usr/lib/systemd/system/cloud-gate.service
#install -d %{buildroot}/%{_datarootdir}/keymasterd/static_files/
#install -p -m 0644 cmd/keymasterd/static_files/u2f-api.js  %{buildroot}/%{_datarootdir}/keymasterd/static_files/u2f-api.js
#install -p -m 0644 cmd/keymasterd/static_files/keymaster-u2f.js  %{buildroot}/%{_datarootdir}/keymasterd/static_files/keymaster-u2f.js
#install -p -m 0644 cmd/keymasterd/static_files/webui-2fa-u2f.js  %{buildroot}/%{_datarootdir}/keymasterd/static_files/webui-2fa-u2f.js
#install -p -m 0644 cmd/keymasterd/static_files/webui-2fa-symc-vip.js  %{buildroot}/%{_datarootdir}/keymasterd/static_files/webui-2fa-symc-vip.js
#install -p -m 0644 cmd/keymasterd/static_files/keymaster.css  %{buildroot}/%{_datarootdir}/keymasterd/static_files/keymaster.css
#install -p -m 0644 cmd/keymasterd/static_files/jquery-1.12.4.patched.min.js %{buildroot}/%{_datarootdir}/keymasterd/static_files/jquery-1.12.4.patched.min.js
install -d %{buildroot}/%{_datarootdir}/cloud-gate/customization_data/templates
install -p -m 0644 cmd/cloud-gate/customization_data/templates/header_extra.tmpl %{buildroot}/%{_datarootdir}/cloud-gate/customization_data/templates/header_extra.tmpl
install -p -m 0644 cmd/cloud-gate/customization_data/templates/footer_extra.tmpl %{buildroot}/%{_datarootdir}/cloud-gate/customization_data/templates/footer_extra.tmpl
install -d %{buildroot}/%{_datarootdir}/cloud-gate/customization_data/web_resources
install -p -m 0644 cmd/cloud-gate/customization_data/web_resources/favicon.ico %{buildroot}/%{_datarootdir}/cloud-gate/customization_data/web_resources/favicon.ico
install -p -m 0644 cmd/cloud-gate/customization_data/web_resources/customization.css %{buildroot}/%{_datarootdir}/cloud-gate/customization_data/web_resources/customization.css

%pre
/usr/bin/getent passwd cloud-gate || useradd -d /var/lib/cloud-gate -s /bin/false -U -r  cloud-gate

%post
mkdir -p /etc/cloud-gate/
mkdir -p /var/lib/cloud-gate
chown cloud-gate /var/lib/cloud-gate
systemctl daemon-reload

%postun
/usr/sbin/userdel cloud-gate
systemctl daemon-reload

%files
#%doc
%{_bindir}/cloud-gate
/usr/lib/systemd/system/cloud-gate.service
#%{_datarootdir}/keymasterd/static_files/*
%config(noreplace) %{_datarootdir}/cloud-gate/customization_data/web_resources/*
%config(noreplace) %{_datarootdir}/cloud-gate/customization_data/templates/*
%changelog


