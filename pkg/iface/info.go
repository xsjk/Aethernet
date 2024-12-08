package iface

/*
#include <winsock2.h>
#include <iptypes.h>
typedef IP_ADAPTER_ADDRESSES AdapterAddresses;
*/
import "C"

import (
	"net"
	"unsafe"

	"Aethernet/pkg/iface/iphlpapi"

	"golang.org/x/sys/windows"
)

type Info struct {
	P *C.AdapterAddresses
}

func (a Info) FriendlyName() string {
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(a.P.FriendlyName)))
}

func (a Info) AdapterName() string {
	return windows.BytePtrToString((*byte)(unsafe.Pointer(a.P.AdapterName)))
}

func (a Info) PcapName() string {
	return "\\Device\\NPF_" + a.AdapterName()
}

func (a Info) Description() string {
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(a.P.Description)))
}

func (a Info) LUID() uint64 {
	return *(*uint64)(unsafe.Pointer(&a.P.Luid[0]))
}

func (a Info) PhysicalAddress() net.HardwareAddr {
	return (*[8]byte)(unsafe.Pointer(&a.P.PhysicalAddress[0]))[:]
}

func (a Info) GetIPv4() (ip net.IP, ipnet *net.IPNet) {
	for p := a.P.FirstUnicastAddress; p != nil; p = p.Next {
		if p.Address.lpSockaddr.sa_family == C.AF_INET {
			ip = net.IPv4(
				byte(p.Address.lpSockaddr.sa_data[2]),
				byte(p.Address.lpSockaddr.sa_data[3]),
				byte(p.Address.lpSockaddr.sa_data[4]),
				byte(p.Address.lpSockaddr.sa_data[5]),
			)
			mask := net.CIDRMask(int(p.OnLinkPrefixLength), 32)
			ipnet = &net.IPNet{
				IP:   ip.Mask(mask),
				Mask: mask,
			}
			return
		}
	}
	return
}

func (a Info) SetIPv4(cidr string) error {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	curip, curipnet := a.GetIPv4()
	if ip.Equal(curip) && ipnet.IP.Equal(curipnet.IP) {
		return nil
	}

	return SetIPv4(a.LUID(), cidr)
}

func SetIPv4(luid uint64, cidr string) error {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	ones, _ := ipnet.Mask.Size()

	row := iphlpapi.InitializeUnicastIpAddressEntry()
	ipv4 := (*C.struct_sockaddr_in)(unsafe.Pointer(&row.Address))
	ipv4.sin_family = C.AF_INET
	copy(ipv4.sin_addr.S_un[:], ip.To4())
	*(*uint8)(&row.OnLinkPrefixLength) = uint8(ones)
	row.DadState = C.IpDadStatePreferred
	*(*uint64)(unsafe.Pointer(&row.InterfaceLuid)) = luid
	err = iphlpapi.CreateUnicastIpAddressEntry(&row)
	return err
}

func GetInfo(friendlyName string) (info Info, err error) {

	adapters, err := iphlpapi.GetAdaptersAddresses(C.AF_UNSPEC, C.GAA_FLAG_INCLUDE_PREFIX)
	if err != nil {
		return
	}

	for _, a := range adapters {
		info.P = (*C.AdapterAddresses)(unsafe.Pointer(&a))
		if info.FriendlyName() == friendlyName {
			return Info{P: info.P}, nil
		}
	}

	err = windows.ERROR_NOT_FOUND
	return

}
