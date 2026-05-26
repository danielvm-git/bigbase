output "public_ip" {
  description = "Reserved public IP address of the Appwrite instance"
  value       = oci_core_public_ip.appwrite.ip_address
}

output "ssh_command" {
  description = "SSH into the instance"
  value       = "ssh ubuntu@${oci_core_public_ip.appwrite.ip_address}"
}

output "appwrite_url" {
  description = "Appwrite console URL (available ~5 min after apply once DNS propagates)"
  value       = "https://${var.appwrite_domain}"
}

output "dns_record" {
  description = "Create this DNS A record before Traefik can issue a TLS certificate"
  value       = "${var.appwrite_domain} → ${oci_core_public_ip.appwrite.ip_address}"
}

output "capacity_status" {
  description = "Ampere A1 capacity status at apply time"
  value       = local.capacity_status
}
