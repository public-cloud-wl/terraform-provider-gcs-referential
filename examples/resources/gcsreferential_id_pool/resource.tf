resource "gcsreferential_id_pool" "example" {
  name       = "examplepoolmaarc"
  start_from = each.value.start
  end_to     = each.value.end
}
