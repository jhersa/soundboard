output "vm_ip" {
  value = google_compute_address.soundboard.address
}

output "ssh_user" {
  value = var.ssh_user
}

output "base_domain" {
  value = var.base_domain
}

output "registry" {
  value = "${google_artifact_registry_repository.images.location}-docker.pkg.dev/${var.gcp_project}/${google_artifact_registry_repository.images.repository_id}"
}

output "media_bucket" {
  value = google_storage_bucket.media.name
}

output "cdn_base_url" {
  value = "https://storage.googleapis.com/${google_storage_bucket.media.name}"
}

output "backup_bucket" {
  value = google_storage_bucket.backups.name
}
