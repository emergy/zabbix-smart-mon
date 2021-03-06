%define debug_package %{nil}
%global __strip /bin/true
%global _dwz_low_mem_die_limit 0

%define go_version 1.9.2

Summary: SMART Monitoring for Zabbix
Name: zabbix-smart-mon
Version: 1.1
Release: 2
License: WTFPL
Group: System Environment/Daemons
Source0: %{name}.src.tar.gz
Source1: %{name}.cron
Source2: %{name}.conf
Source3: gopkgs.tar.gz
ExclusiveArch: x86_64
Requires: zabbix-sender >= 2.4, smartmontools
BuildRequires: golang, gcc

%description
SMART Monitoring for Zabbix

%prep
tar -C ${RPM_BUILD_DIR} -xzf %{SOURCE3}
mkdir -p ${RPM_BUILD_DIR}/go/src
tar xzvf %{SOURCE0} -C ${RPM_BUILD_DIR}/go/src/

%build
export GOARCH="amd64"
export GOROOT="/usr/local/go"
export GOTOOLDIR="/usr/local/go/pkg/tool/linux_amd64"
export GOPATH="${RPM_BUILD_DIR}/go"
export PATH="$PATH:$GOROOT/bin:$GOPATH/bin"

cd ${RPM_BUILD_DIR}/go/src/%{name}
go build -a -ldflags "-B 0x$(head -c20 /dev/urandom|od -An -tx1|tr -d ' \n')" -v -x .

%install
install -d %{buildroot}%{_bindir}
#cp -f ${RPM_BUILD_DIR}/%{name} %{buildroot}%{_bindir}/%{name}
cp -f ${RPM_BUILD_DIR}/go/src/%{name}/%{name} %{buildroot}%{_bindir}/%{name}
install -d %{buildroot}/var/log/%{name}
install -d %{buildroot}/etc/cron.d
install -d %{buildroot}/etc/logrotate.d
cp -f %{SOURCE1}  %{buildroot}/etc/cron.d/zabbix-smart-mon
cp -f %{SOURCE2}  %{buildroot}/etc/logrotate.d/%{name}

%clean
rm -rf %{buildroot}

%files
%defattr(-,root,root,-)
%{_bindir}/%{name}
%dir /var/log/%{name}
/etc/cron.d/zabbix-smart-mon
/etc/logrotate.d/%{name}

%post
sed -i "s/^0/`hostid | perl -0ne 'print hex($_) % 60'`/" /etc/cron.d/zabbix-smart-mon

%changelog
* Thu Nov 09 2017 Alex Emergy <alex.emergy@gmail.com> - 1.0
- Initial RPM release for EL7.

