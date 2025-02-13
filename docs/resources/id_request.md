---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "gcsreferential_id_request Resource - terraform-provider-gcsreferential"
subcategory: ""
description: |-
  This resource allow you to request and id from an id_pool
---

# gcsreferential_id_request (Resource)

This resource allow you to request and id from an id_pool

## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `id` (String) The terraform id of the resource
- `pool` (String) The name of the pool, to make the id_request on. If you change it, the id_request will be destroyed and recreate

### Read-Only

- `requested_id` (Number) The requested id from the pool, a free one that will be reserved for this resource


