Name:     link-share
Version: %{?version:%version}%{?!version:0.0.0}
Release: %{?build_number:%build_number}%{?!build_number:1}
Summary:  code-ointment link-share
License:  MIT
URL:      https://github.com/code-ointment/
Source:   link-share.tar.gz

%changelog
* Thu Jan 9 2025 ssanders
- initial

%description
Share VPN links on a home network

%prep
tar -zxf $RPM_SOURCE_DIR/%{name}.tar.gz

#------------------------------------------------
# build (happens on build machine)
#
# Compile the code
#------------------------------------------------
%build

cd $RPM_BUILD_DIR
go version
make 

#------------------------------------------------
#
#------------------------------------------------
%install

rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/var/log/code-ointment/link-share/
mkdir -p $RPM_BUILD_ROOT/opt/code-ointment/link-share/bin
mkdir -p $RPM_BUILD_ROOT/opt/code-ointment/link-share/etc

cp $RPM_BUILD_DIR/bin/link-share $RPM_BUILD_ROOT/opt/code-ointment/link-share/bin
cp $RPM_BUILD_DIR/scripts/link-share.sh $RPM_BUILD_ROOT/opt/code-ointment/link-share/bin
chmod 755 $RPM_BUILD_ROOT/opt/code-ointment/link-share/bin/link-share.sh

cp $RPM_BUILD_DIR/etc/link-share.service $RPM_BUILD_ROOT/opt/code-ointment/link-share/etc

%clean
rm -rf $RPM_BUILD_ROOT

%files
%defattr(-,root,root,-)
  %dir /opt/code-ointment/link-share/bin
  %dir /opt/code-ointment/link-share/etc
  %dir /var/log/code-ointment/link-share
  /opt/code-ointment/link-share/bin/link-share
  /opt/code-ointment/link-share/bin/link-share.sh
  /opt/code-ointment/link-share/etc/link-share.service

%pre
# Stuff that needs to execute just before package is to be installed
# If this is a new install (not an upgrade)
if [ $1 -eq 1 ] ; then
  echo "pre install"
else
  # upgrade scenario
  systemctl stop link-share >/dev/null 2>&1
  # Return 0 exit status regardless of above commands
  true
fi


%post

install_service() {
  cp /opt/code-ointment/link-share/etc/link-share.service \
    /usr/lib/systemd/system

  systemctl daemon-reload
  systemctl enable link-share
  systemctl start link-share
}

if [ $1 -eq 1 ]; then
  # Install case
  install_service
else 
  # Update case
  systemctl stop link-share
  install_service
fi

true

%preun
# Stuff that needs to execute just prior to the package being removed
if [ $1 -eq 0 ] ; then
  # uninstall scenario
  systemctl stop link-share
  # Return 0 exit status regardless of above commands
  true
fi


%postun
# Stuff that needs to execute after package has been removed
if [ $1 -eq 0 ] ; then
  # uninstall scenario
  systemctl daemon-reload
  if [ -d /var/log/code-ointment/link-share ]; then
    rm -rf /var/log/code-ointment/link-share
  fi
  true
fi

