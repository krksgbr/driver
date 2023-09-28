{ lib, stdenv, fetchFromGitHub, pkg-config, autoconf-archive, autoreconfHook, python3, perl }:

stdenv.mkDerivation rec {
  name = "pcsclite-static-${version}";
  version = "1.9.5";

  src = fetchFromGitHub {
    owner = "LudovicRousseau";
    repo = "PCSC";
    rev = "${version}";
    sha256 = "sha256-ZgYwI/A0dxRYeLxteFG5fiArzJ292q7oaD9uAszSIZo=";
  };

  patches = [ ./no-dropdir-literals.patch ];

  preConfigurePhases = "bootStrap";

  bootStrap = ''
    ./bootstrap
  '';

  configureFlags = [
    # The OS should care on preparing the drivers into this location
    "--enable-usbdropdir=/var/lib/pcsc/drivers"
    "--enable-confdir=/etc"
    "--enable-ipcdir=/run/pcscd"
    # disable unnecessary stuff
    "--disable-libudev"
    "--disable-libusb"
    "--disable-libsystemd"
    "--disable-documentation"
    # enable static linking
    "--enable-static"
  ] ++ lib.optional stdenv.isLinux
         "--with-systemdsystemunitdir=\${out}/etc/systemd/system";

  postConfigure = ''
    sed -i -re '/^#define *PCSCLITE_HP_DROPDIR */ {
      s/(DROPDIR *)(.*)/\1(getenv("PCSCLITE_HP_DROPDIR") ? : \2)/
    }' config.h
  '';

  enableParallelBuilding = true;
  nativeBuildInputs = [ autoreconfHook autoconf-archive pkg-config perl ];
  buildInputs = [ python3 ];
    #++ lib.optionals polkitSupport [ dbus polkit ];

  meta = with lib; {
    description = "Middleware to access a smart card using SCard API (PC/SC)";
    homepage = http://pcsclite.alioth.debian.org/;
    license = licenses.bsd3;
    maintainers = with maintainers; [ viric wkennington ];
    platforms = with platforms; unix;
  };
}
