data "oci_identity_availability_domains" "ads" {
  compartment_id = local.compartment_id
}

# Capacity check — read-only resource for validation
resource "oci_core_compute_capacity_report" "ampere" {
  availability_domain = data.oci_identity_availability_domains.ads.availability_domains[0].name
  compartment_id      = local.compartment_id

  shape_availabilities {
    instance_shape = "VM.Standard.A1.Flex"
    instance_shape_config {
      ocpus         = var.instance_ocpus
      memory_in_gbs = var.instance_memory_in_gbs
    }
  }
}

locals {
  capacity_status = oci_core_compute_capacity_report.ampere.shape_availabilities[0].availability_status
}

# Latest Ubuntu 22.04 ARM image for the chosen region
data "oci_core_images" "ubuntu_arm" {
  compartment_id           = local.compartment_id
  operating_system         = "Canonical Ubuntu"
  operating_system_version = "22.04"
  shape                    = "VM.Standard.A1.Flex"
  sort_by                  = "TIMECREATED"
  sort_order               = "DESC"
  state                    = "AVAILABLE"
}

resource "oci_core_instance" "appwrite" {
  compartment_id      = local.compartment_id
  availability_domain = data.oci_identity_availability_domains.ads.availability_domains[0].name
  display_name        = "appwrite"
  shape               = "VM.Standard.A1.Flex"

  shape_config {
    ocpus         = var.instance_ocpus
    memory_in_gbs = var.instance_memory_in_gbs
  }

  source_details {
    source_type             = "image"
    source_id               = data.oci_core_images.ubuntu_arm.images[0].id
    boot_volume_size_in_gbs = var.boot_volume_size_in_gbs
  }

  create_vnic_details {
    subnet_id        = oci_core_subnet.appwrite.id
    assign_public_ip = false
    display_name     = "appwrite-vnic"
    hostname_label   = "appwrite"
  }

  metadata = {
    ssh_authorized_keys = var.ssh_public_key
    user_data = base64encode(templatefile("${path.module}/cloud-init.yaml.tpl", {
      appwrite_domain = var.appwrite_domain
    }))
  }

  lifecycle {
    precondition {
      condition     = local.capacity_status == "AVAILABLE"
      error_message = "Ampere A1 capacity is ${local.capacity_status} in ${data.oci_identity_availability_domains.ads.availability_domains[0].name}. Try another region or a smaller shape."
    }
  }
}

# Primary private IP of the instance's VNIC — needed to attach a reserved public IP
data "oci_core_vnic_attachments" "appwrite" {
  compartment_id      = local.compartment_id
  availability_domain = oci_core_instance.appwrite.availability_domain
  instance_id         = oci_core_instance.appwrite.id
}

data "oci_core_vnic" "appwrite" {
  vnic_id = data.oci_core_vnic_attachments.appwrite.vnic_attachments[0].vnic_id
}

data "oci_core_private_ips" "appwrite" {
  vnic_id = data.oci_core_vnic.appwrite.id
}

# Reserved public IP — survives instance rebuilds
resource "oci_core_public_ip" "appwrite" {
  compartment_id = local.compartment_id
  lifetime       = "RESERVED"
  display_name   = "appwrite-reserved-ip"
  private_ip_id  = data.oci_core_private_ips.appwrite.private_ips[0].id
}
