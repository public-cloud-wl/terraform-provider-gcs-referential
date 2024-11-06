resource "gcsreferential_id_pool" "example" {
  name       = "examplepoolmaarc"
  start_from = each.value.start
  end_to     = each.value.end
}
resource "gcsreferential_id_request" "example" {
  pool = gcsreferential_id_pool.example.name
  id   = "maarc-id"
}
resource "gcsreferential_id_request" "example2" {
  for_each = toset([for i in range(1, 13) : "maarc-${i}"])
  pool     = gcsreferential_id_pool.example.name
  id       = each.key
}
