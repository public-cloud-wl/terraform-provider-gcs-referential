# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "gcsreferential_id_pool" "example" {
  for_each   = local.p
  name       = "examplepoolmaarc${each.key}"
  start_from = each.value.start
  end_to     = each.value.end
}
resource "gcsreferential_id_request" "example" {
  for_each = toset([for i in range(1, 13) : "maarc-${i}"])
  pool     = gcsreferential_id_pool.example["1"].name
  id       = each.key
}
