# Terraform Provider Cidr-Reservator (Terraform Plugin SDK)

Terraform Provider for reserving Cidr Ranges in a central location (currently only GCS Buckets are supported).
When reserving a new Cidr within a Base-Cidr the next available Cidr is calculated. Possible gaps are filled if possible. If the Base-Cidr is exhausted, an error is thrown.


