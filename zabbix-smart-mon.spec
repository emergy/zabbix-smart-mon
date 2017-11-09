%define debug_package %{nil}
%global __strip /bin/true
%global _dwz_low_mem_die_limit 0

%define go_version 1.9.2

Summary: SMART Monitoring for Zabbix
Name: zabbix-smart-mon
Version: 1.0
Release: 1
License: WTFPL
Group: System Environment/Daemons
Source0: %{name}.go
Source2: go%{go_version}.linux-amd64.tar.gz
Source3: gopkgs.tar.gz
ExclusiveArch: x86_64
Requires: zabbix-sender >= 2.4

%description
SMART Monitoring for Zabbix

%prep
mkdir -p ${RPM_BUILD_DIR}/usr/local
tar -C ${RPM_BUILD_DIR}/usr/local -xzf %{SOURCE2}
tar -C ${RPM_BUILD_DIR}/usr/local -xzf %{SOURCE3}
mkdir -p ${RPM_BUILD_DIR}/goprj/src/%{name}
cp -f %{SOURCE0} ${RPM_BUILD_DIR}/goprj/src/%{name}

%build
export GOARCH="amd64"
export GOROOT="${RPM_BUILD_DIR}/usr/local/go"
export GOTOOLDIR="${RPM_BUILD_DIR}/usr/local/go/pkg/tool/linux_amd64"
export GOPATH="${RPM_BUILD_DIR}/goprj"
export PATH="$PATH:$GOROOT/bin"

go build -a -ldflags "-B 0x$(head -c20 /dev/urandom|od -An -tx1|tr -d ' \n')" -v -x %{name}

%install
install -d %{buildroot}%{_bindir}
cp -f ${RPM_BUILD_DIR}/%{name} %{buildroot}%{_bindir}/%{name}
install -d %{buildroot}/var/log/%{name}

%clean
rm -rf %{buildroot}

%files
%defattr(-,root,root,-)
%{_bindir}/%{name}
%dir /var/log/%{name}

%changelog
* Thu Nov 09 2017 Alex Emergy <alex.emergy@gmail.com> - 1.0
- Initial RPM release for EL7.

