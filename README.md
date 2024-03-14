# AltDSS-Go

Golang (CGo) bindings for AltDSS/DSS C-API. Tries to mimic the organization of the official OpenDSS COM classes, within the scope of Go's language features.

This is a new project, initially targeting Linux. Other platforms will be added later. There is no platform-specific code nor dependencies, but due to use of CGo, building instructions are specific to each platform.

## Tentative instructions

While proper configuration and scripts are not in place:

```bash
git clone https://github.com/dss-extensions/electricdss-tst # for sample files
git clone https://github.com/dss-extensions/altdss-go
cd altdss-go
wget -qO- https://github.com/dss-extensions/dss_capi/releases/download/0.14.3/dss_capi_0.14.3_linux_x64.tar.gz | tar zxv
export CPATH=`pwd`/dss_capi/include/
mv dss_capi/lib/linux_x64/*.so .
export LIBRARY_PATH=`pwd`
go build ./altdss # this takes some time
go build -o . ./examples/simple.go
./simple
```

