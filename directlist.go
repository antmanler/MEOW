package main

import (
	"net"
	"os"
	"strings"
	"sync"

	"github.com/cyfdecyf/bufio"
)

type DirectList struct {
	Domain map[string]DomainType
	sync.RWMutex
}

type DomainType byte

const (
	domainTypeUnknown DomainType = iota
	domainTypeDirect
	domainTypeProxy
	domainTypeReject
)
const maxHostPart = 5

func newDirectList() *DirectList {
	return &DirectList{
		Domain: map[string]DomainType{},
	}
}

func charIndex(host string, c byte) []int {
	indices := make([]int, 0, maxHostPart+1)
	n := len(host)

	if n <= 1 {
		return indices
	}

	for i := 0; i < n-1; i++ {
		if host[i] == c {
			indices = append(indices, i)
		}
	}
	return indices
}

func domainSearch(domain map[string]DomainType, host string, isIP bool) (domainType DomainType, ok bool) {
	if domainType, ok = domain[host]; !ok && !isIP {
		indices := charIndex(host, '.')
		n := len(indices)
		if n > maxHostPart {
			indices = indices[n-maxHostPart:]
			n = maxHostPart
		}

		for i := 0; i < n; i++ {
			suffix := host[indices[i]+1:]
			if domainType, ok = domain[suffix]; ok {
				break
			}
		}
	}

	if parentProxy.empty() {
		if domainType == domainTypeReject {
			return domainTypeReject, true
		} else {
			return domainTypeDirect, true
		}
	}
	return
}

func (directList *DirectList) shouldDirect(url *URL) (domainType DomainType) {
	debug.Printf("judging host: %s", url.Host)

	if url.Domain == "" { // simple host or private ip
		return domainTypeDirect
	}

	isIP, isPrivate := hostIsIP(url.Host)
	if v, ok := domainSearch(directList.Domain, url.Host, isIP); ok {
		return v
	}

	if !config.JudgeByIP {
		return domainTypeProxy
	}

	var ip string
	if isIP {
		if isPrivate {
			directList.add(url.Host, domainTypeDirect)
			return domainTypeDirect
		}
		ip = url.Host
	} else {
		hostIPs, err := net.LookupIP(url.Host)
		if err != nil {
			errl.Printf("error looking up host ip %s, err %s", url.Host, err)
			return domainTypeProxy
		}
		ip = hostIPs[0].String()
	}

	if ipShouldDirect(ip) {
		directList.add(url.Host, domainTypeDirect)
		return domainTypeDirect
	} else {
		directList.add(url.Host, domainTypeProxy)
		return domainTypeProxy
	}
}

func (directList *DirectList) add(host string, domainType DomainType) {
	directList.Lock()
	defer directList.Unlock()
	directList.Domain[host] = domainType
}

func (directList *DirectList) GetDirectList() []string {
	lst := make([]string, 0)
	for site, domainType := range directList.Domain {
		if domainType == domainTypeDirect {
			lst = append(lst, site)
		}
	}
	return lst
}

var directList = newDirectList()

func initDomainList(domainListFile string, domainType DomainType) {
	var err error
	if err = isFileExists(domainListFile); err != nil {
		return
	}
	f, err := os.Open(domainListFile)
	if err != nil {
		errl.Println("Error opening domain list:", err)
		return
	}
	defer f.Close()

	directList.Lock()
	defer directList.Unlock()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain == "" {
			continue
		}
		debug.Printf("Loaded domain %s as type %v", domain, domainType)
		directList.Domain[domain] = domainType
	}
	if scanner.Err() != nil {
		errl.Printf("Error reading domain list %s: %v\n", domainListFile, scanner.Err())
	}
}
