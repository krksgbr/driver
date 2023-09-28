{ lib, stdenv, fetchFromGitHub, pkgconfig, autoconf, automake, libtool, flex, python3, perl
, IOKit ? null }:

stdenv.mkDerivation rec {
  name = "pcsclite-static-${version}";
  version = "1.8.23";

  src = fetchFromGitHub {
    owner = "LudovicRousseau";
    repo = "PCSC";
    rev = "pcsc-${version}";
    sha256 = "0pahf0s9zljfi0byi1s78y40k918g1prc37mp4gr4hzb7jff0zw4";
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

  nativeBuildInputs = [ autoconf automake libtool flex pkgconfig perl python3 ];

  meta = with lib; {
    description = "Middleware to access a smart card using SCard API (PC/SC)";
    homepage = http://pcsclite.alioth.debian.org/;
    license = licenses.bsd3;
    maintainers = with maintainers; [ viric wkennington ];
    platforms = with platforms; unix;
  };
}
