package iphlpapi

/*
#include <winsock2.h>
#include <ws2ipdef.h>
#include <iphlpapi.h>
#include <netioapi.h>
#include <iptypes.h>
typedef IP_ADAPTER_ADDRESSES AdapterAddresses;
*/
import "C"
import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func InitializeUnicastIpAddressEntry() (row C.MIB_UNICASTIPADDRESS_ROW) {
	initializeUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&row)))
	return
}

func CreateUnicastIpAddressEntry(row *C.MIB_UNICASTIPADDRESS_ROW) (err error) {
	ret, _, _ := createUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(row)))
	err = windows.Errno(ret)
	if err == windows.ERROR_SUCCESS {
		err = nil
	}
	return
}

func GetAdaptersAddresses(family C.ULONG, flags C.ULONG) (addresses []C.AdapterAddresses, err error) {
	bufferSize := 128

	for {
		buffer := make([]byte, bufferSize)
		pAddresses := uintptr(unsafe.Pointer(&buffer[0]))

		ret, _, _ := getAdaptersAddresses.Call(
			uintptr(family),
			uintptr(flags),
			uintptr(0),
			pAddresses,
			uintptr(unsafe.Pointer(&bufferSize)),
		)
		switch ret {
		case C.ERROR_BUFFER_OVERFLOW:
			bufferSize *= 2
		case C.ERROR_SUCCESS:
			err = nil
			addresses = make([]C.AdapterAddresses, 0)
			ptr := uintptr(unsafe.Pointer(&buffer[0]))
			for ptr != 0 {
				adapterPtr := (*C.AdapterAddresses)(unsafe.Pointer(ptr))
				addresses = append(addresses, *adapterPtr)
				ptr = uintptr(unsafe.Pointer(adapterPtr.Next))
			}
			return
		default:
			err = windows.Errno(ret)
			return
		}
	}
}
