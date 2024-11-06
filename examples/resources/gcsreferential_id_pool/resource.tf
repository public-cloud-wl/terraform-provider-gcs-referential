# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "gcsreferential_id_pool" "example" {
  for_each   = local.p
  name       = "examplepoolmaarc${each.key}"
  start_from = each.value.start
  end_to     = each.value.end
}
