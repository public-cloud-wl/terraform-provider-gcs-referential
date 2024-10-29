// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// providerFactories are used to instantiate a provider during acceptance testing.
// The factory function will be invoked for every Terraform CLI command executed
// to create a provider server to which the CLI can reattach.
var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"gcsreferential": providerserver.NewProtocol6WithError(New("dev")()),
}

func TestProvider(t *testing.T) {
	if err := New("dev")(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
