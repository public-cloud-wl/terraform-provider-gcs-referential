package cidrCalculator

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/apparentlymart/go-cidr/cidr"
	"net"
	"sort"
	"strconv"
	"strings"
)

type cidrCalculator struct {
	currentSubnets       *map[string]string
	prefixLength         int8
	baseCidrRange        string
	baseCidrPrefixLength int8
	baseIPNet            *net.IPNet
}

func New(currentSubnets *map[string]string, prefixLength int8, baseCidrRange string) (cidrCalculator, error) {
	baseCidrPrefixLength, err := strconv.ParseInt(strings.Split(baseCidrRange, "/")[1], 10, 8)
	if err != nil {
		return cidrCalculator{}, err
	}
	_, baseIPNet, _ := net.ParseCIDR(baseCidrRange)
	return cidrCalculator{currentSubnets, prefixLength, baseCidrRange, int8(baseCidrPrefixLength) - 1, baseIPNet}, nil
}

// TODO: implement!
func (c cidrCalculator) GetNextNetmask() (string, error) {
	if c.prefixLength > 32 || c.prefixLength < 0 {
		return "", errors.New("prefixLength must be an integer between 0 and 32")
	}
	ipNets := make([]*net.IPNet, 0, len(*c.currentSubnets))
	for _, value := range *c.currentSubnets {
		_, ipNet, err := net.ParseCIDR(value)
		if err != nil {
			return "", err
		}
		ipNets = append(ipNets, ipNet)
	}
	sort.Slice(ipNets, func(i, j int) bool {
		return bytes.Compare(ipNets[i].IP, ipNets[j].IP) > 0
	})
	//ipNetsCidr := make([]string, 0, len(*currentSubnets))
	//for _, ipNet := range ipNets {
	//	ipNetsCidr = append(ipNetsCidr, ipNet.String())
	//}
	var nextIPNet *net.IPNet
	firstCidrSubnet, err := cidr.Subnet(c.baseIPNet, int(c.prefixLength-c.baseCidrPrefixLength-1), 0)
	if err != nil {
		return "", err
	}
	if len(ipNets) == 0 || (cidr.VerifyNoOverlap(append(ipNets, firstCidrSubnet), c.baseIPNet) == nil) {
		nextIPNet = firstCidrSubnet
	} else {
		nextIPNet, err = c.recursivelyFindNextNetmask(&ipNets, c.prefixLength, false)
	}
	if err != nil {
		return "", err
	}
	//nextNetBaseIP, _, _ := net.ParseCIDR(nextNetmask)
	if !c.baseIPNet.Contains(nextIPNet.IP) {
		return "", fmt.Errorf("baseCidrRange %s is exhausted!", c.baseCidrRange)
	}
	err = cidr.VerifyNoOverlap(append(ipNets, nextIPNet), c.baseIPNet)
	if err != nil {
		return "", fmt.Errorf("The produced subnet overlaps with existing subnets! This should not happen and is a bug. Please report it!")
	}
	return nextIPNet.String(), nil
}

// This algorithm first tries to find the next subnet and fill "gaps" as good as possible. Therefore it starts to search at an already reserved subnet with equal or smaller prefix size. It afterwards continues with bigger prefix sizes.
func (c cidrCalculator) recursivelyFindNextNetmask(ipNets *[]*net.IPNet, searchPrefixLength int8, doneWithBiggerEqualPrefix bool) (*net.IPNet, error) {
	if searchPrefixLength <= c.baseCidrPrefixLength {
		lastSubnet := (*ipNets)[0]
		nextIPNet, exhausted := cidr.NextSubnet(lastSubnet, int(c.prefixLength))
		if exhausted {
			return nil, fmt.Errorf("Maximum IP exhausted!")
		}
		return nextIPNet, nil
	}
	_, maskNet, _ := net.ParseCIDR(fmt.Sprintf("0.0.0.0/%d", searchPrefixLength))
	mask := maskNet.Mask
	var previousRunSubnet *net.IPNet
	for index, ipNet := range *ipNets {
		compare := bytes.Compare(mask, ipNet.Mask)
		if compare <= 0 && !doneWithBiggerEqualPrefix {
			nextSubnet, noOverlap, err := c.getNextSubnetVerifyNoOverlap(index, ipNets, previousRunSubnet)
			if err != nil {
				return nil, err
			}
			if noOverlap {
				return nextSubnet, nil
			}
			previousRunSubnet = nextSubnet
		} else if doneWithBiggerEqualPrefix && compare == 0 {
			nextSubnet, noOverlap, err := c.getNextSubnetVerifyNoOverlap(index, ipNets, previousRunSubnet)
			if err != nil {
				return nil, err
			}
			if noOverlap {
				return nextSubnet, nil
			}
		}
	}
	return c.recursivelyFindNextNetmask(ipNets, searchPrefixLength-1, true)
}

func (c cidrCalculator) getNextSubnetVerifyNoOverlap(index int, ipNets *[]*net.IPNet, previousRunSubnet *net.IPNet) (*net.IPNet, bool, error) {
	var previousSubnet *net.IPNet
	if index > 0 {
		previousSubnet = (*ipNets)[index-1]
	}
	calculatedNextSubnet, exhausted := cidr.NextSubnet((*ipNets)[index], int(c.prefixLength))
	if exhausted {
		return nil, false, fmt.Errorf("Maximum IP exhausted!")
	}
	if !c.baseIPNet.Contains(calculatedNextSubnet.IP) {
		return calculatedNextSubnet, false, nil
	}
	if previousSubnet != nil {
		if !calculatedNextSubnet.Contains(previousSubnet.IP) && !previousSubnet.Contains(calculatedNextSubnet.IP) {
			if previousRunSubnet == nil || !previousRunSubnet.Contains(calculatedNextSubnet.IP) {
				return calculatedNextSubnet, true, nil
			}
		}
	} else {
		// in this case there is no previous subnet in the list; as this iteration is reverse over the list, this means, we already have the last reserved subnet
		return calculatedNextSubnet, true, nil
	}
	return calculatedNextSubnet, false, nil
}
