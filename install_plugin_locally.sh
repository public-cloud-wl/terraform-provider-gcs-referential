#!/bin/bash

providerName=terraform-provider-gcsreferential

rm $providerName || true
pluginDir=/home/admnet/git/worldline/ccc/terraform/projects/maarc-tests/.terraform/providers/registry.terraform.io/public-cloud-wl/gcsreferential/0.0.8/linux_amd64/
rm $pluginDir/$providerName* || true
go build
mkdir -p $pluginDir
cp $providerName $pluginDir/
