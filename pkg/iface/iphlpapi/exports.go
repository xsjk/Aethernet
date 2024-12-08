package iphlpapi

import "golang.org/x/sys/windows"

var (
	iphlpapi                        = windows.NewLazySystemDLL("iphlpapi.dll")
	initializeUnicastIpAddressEntry = iphlpapi.NewProc("InitializeUnicastIpAddressEntry")
	createUnicastIpAddressEntry     = iphlpapi.NewProc("CreateUnicastIpAddressEntry")
	getAdaptersAddresses            = iphlpapi.NewProc("GetAdaptersAddresses")
)
