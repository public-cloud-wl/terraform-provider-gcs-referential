package cidrCalculator

import (
	"fmt"
	"testing"
)

type TestData struct {
	currentSubnets *map[string]string
	prefixLength   int8
	baseCidrRange  string
}

func initTestData() *TestData {
	return &TestData{
		currentSubnets: &map[string]string{"test1": "10.116.0.16/29", "test4": "10.116.3.0/24", "test5": "10.116.8.0/22", "test6": "10.116.4.0/22", "foo": "10.116.2.0/24", "tralala": "10.116.1.0/24", "tut": "10.116.0.8/29", "fioo": "10.116.0.32/28", "tutut": "10.119.0.0/16"},
		prefixLength:   22,
		baseCidrRange:  "10.116.0.0/14",
	}
}

func TestCorrectNextCidr(t *testing.T) {
	testData := initTestData()
	cidrCalculator, err := New(testData.currentSubnets, testData.prefixLength, testData.baseCidrRange)
	if err != nil {
		t.Fatal(err)
	}
	netmask, err := cidrCalculator.GetNextNetmask()
	if err != nil {
		t.Fatal(err)
	}
	if netmask != "10.116.12.0/22" {
		t.Fatalf("Unexpected value for next netmask %s", netmask)
	}
}

func TestCorrectNextCidr16StillFits(t *testing.T) {
	testData := initTestData()
	testData.prefixLength = 16
	cidrCalculator, err := New(testData.currentSubnets, testData.prefixLength, testData.baseCidrRange)
	if err != nil {
		t.Fatal(err)
	}
	netmask, err := cidrCalculator.GetNextNetmask()
	if err != nil {
		t.Fatal(err)
	}
	if netmask != "10.117.0.0/16" {
		t.Fatalf("Unexpected value for next netmask %s", netmask)
	}
}

func TestBaseCidrExhausted(t *testing.T) {
	testData := initTestData()
	expected := fmt.Sprintf("baseCidrRange %s is exhausted!", testData.baseCidrRange)
	testData.prefixLength = 15
	theCidrCalculator, err := New(testData.currentSubnets, testData.prefixLength, testData.baseCidrRange)
	if err != nil {
		t.Fatal(err)
	}
	next := ""
	next, err = theCidrCalculator.GetNextNetmask()
	println(next)
	if err == nil {
		t.Fatal("There should be an error when Cidr Range is exhausted!!!")
	}
	if err.Error() != expected {
		t.Fatalf("The error message does not match: Expected %s, Got %s", expected, err.Error())
	}
}
