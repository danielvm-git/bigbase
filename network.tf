resource "oci_core_vcn" "appwrite" {
  compartment_id = local.compartment_id
  cidr_blocks    = ["10.0.0.0/16"]
  display_name   = "appwrite-vcn"
  dns_label      = "appwrite"
}

resource "oci_core_internet_gateway" "appwrite" {
  compartment_id = local.compartment_id
  vcn_id         = oci_core_vcn.appwrite.id
  display_name   = "appwrite-igw"
  enabled        = true
}

resource "oci_core_route_table" "appwrite" {
  compartment_id = local.compartment_id
  vcn_id         = oci_core_vcn.appwrite.id
  display_name   = "appwrite-rt"

  route_rules {
    network_entity_id = oci_core_internet_gateway.appwrite.id
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
  }
}

resource "oci_core_security_list" "appwrite" {
  compartment_id = local.compartment_id
  vcn_id         = oci_core_vcn.appwrite.id
  display_name   = "appwrite-sl"

  egress_security_rules {
    protocol    = "all"
    destination = "0.0.0.0/0"
  }

  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 22
      max = 22
    }
  }

  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 80
      max = 80
    }
  }

  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 443
      max = 443
    }
  }
}

resource "oci_core_subnet" "appwrite" {
  compartment_id    = local.compartment_id
  vcn_id            = oci_core_vcn.appwrite.id
  cidr_block        = "10.0.1.0/24"
  display_name      = "appwrite-subnet"
  dns_label         = "appwrite"
  route_table_id    = oci_core_route_table.appwrite.id
  security_list_ids = [oci_core_security_list.appwrite.id]
}
