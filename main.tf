terraform {
  required_version = ">= 1.5"
  required_providers {
    oci = {
      source  = "oracle/oci"
      version = "~> 6.0"
    }
  }

  # Uncomment to store state in OCI Object Storage (recommended for production)
  # backend "s3" {
  #   bucket                      = "terraform-state"
  #   key                         = "appwrite/terraform.tfstate"
  #   region                      = "sa-saopaulo-1"
  #   endpoint                    = "https://<tenancy-namespace>.compat.objectstorage.sa-saopaulo-1.oraclecloud.com"
  #   shared_credentials_file     = "/dev/null"
  #   skip_credentials_validation = true
  #   skip_metadata_api_check     = true
  #   skip_region_validation      = true
  #   force_path_style            = true
  # }
}

provider "oci" {
  tenancy_ocid     = var.tenancy_ocid
  user_ocid        = var.user_ocid
  fingerprint      = var.fingerprint
  private_key_path = var.private_key_path
  region           = var.region
}

locals {
  compartment_id = var.compartment_id != "" ? var.compartment_id : var.tenancy_ocid
}
