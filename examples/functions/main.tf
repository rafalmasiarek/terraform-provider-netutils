terraform {
  required_providers {
    netutils = {
      source = "rafalmasiarek/netutils"
    }
  }
}

provider "netutils" {}

output "normalized" {
  value = provider::netutils::ip_normalize("2001:0DB8::0001")
}

output "classification" {
  value = provider::netutils::ip_classify("100.64.0.1")
}

output "in_cidr" {
  value = provider::netutils::ip_in_cidr("10.0.1.5", "10.0.0.0/16")
}

output "contains" {
  value = provider::netutils::cidr_contains("10.0.0.0/8", "10.1.0.0/16")
}

output "overlap" {
  value = provider::netutils::cidr_overlap("10.0.0.0/16", "10.0.128.0/17")
}
