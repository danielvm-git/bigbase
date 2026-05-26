variable "tenancy_ocid" {
  description = "OCID of the OCI tenancy"
  type        = string
}

variable "user_ocid" {
  description = "OCID of the OCI user"
  type        = string
}

variable "fingerprint" {
  description = "Fingerprint of the OCI API signing key"
  type        = string
}

variable "private_key_path" {
  description = "Absolute path to the OCI API private key PEM file"
  type        = string
  default     = "~/.oci/oci_api_key.pem"
}

variable "region" {
  description = "OCI region — switch to sa-vinhedo-1 once subscribed"
  type        = string
  default     = "sa-saopaulo-1"
}

variable "compartment_id" {
  description = "OCID of the compartment (defaults to root tenancy if empty)"
  type        = string
  default     = ""
}

variable "ssh_public_key" {
  description = "SSH public key content to authorize on the instance"
  type        = string
}

variable "appwrite_domain" {
  description = "Domain pointing to this server (e.g. appwrite.example.com). Must have an A record → public IP after apply."
  type        = string
}

variable "instance_ocpus" {
  description = "OCPUs for the Ampere A1 instance. Always Free allows up to 4 total across all A1 instances."
  type        = number
  default     = 4
}

variable "instance_memory_in_gbs" {
  description = "RAM in GB for the Ampere A1 instance. Always Free allows up to 24 GB total."
  type        = number
  default     = 24
}

variable "boot_volume_size_in_gbs" {
  description = "Boot volume size in GB. Always Free includes up to 200 GB total block storage."
  type        = number
  default     = 50
}
