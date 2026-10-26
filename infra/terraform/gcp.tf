locals {
  admin_origin = "https://admin.${var.base_domain}"
}

resource "google_project_service" "services" {
  for_each           = toset(["compute.googleapis.com", "storage.googleapis.com", "iam.googleapis.com", "artifactregistry.googleapis.com"])
  service            = each.value
  disable_on_destroy = false
}

resource "random_id" "suffix" {
  byte_length = 3
}

# ---- VM ----

resource "google_compute_address" "soundboard" {
  name       = "soundboard"
  depends_on = [google_project_service.services]
}

# Cuenta de la VM: la usan la API (medios), los respaldos y Docker (Artifact Registry).
resource "google_service_account" "vm" {
  account_id   = "soundboard-vm"
  display_name = "Soundboard VM"
  depends_on   = [google_project_service.services]
}

resource "google_compute_instance" "soundboard" {
  name         = "soundboard"
  machine_type = "e2-micro"
  tags         = ["soundboard"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
      size  = 30
      type  = "pd-standard"
    }
  }

  network_interface {
    network = "default"
    access_config {
      nat_ip = google_compute_address.soundboard.address
    }
  }

  metadata = {
    ssh-keys = "${var.ssh_user}:${trimspace(file(pathexpand(var.ssh_public_key_path)))}"
  }

  service_account {
    email  = google_service_account.vm.email
    scopes = ["cloud-platform"]
  }

  allow_stopping_for_update = true
}

resource "google_compute_firewall" "web" {
  name          = "soundboard-web"
  network       = "default"
  target_tags   = ["soundboard"]
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "tcp"
    ports    = ["80", "443"]
  }
  depends_on = [google_project_service.services]
}

# Túnel WireGuard hacia la Raspberry Pi de CastBerry (la Pi inicia la conexión).
resource "google_compute_firewall" "wireguard" {
  name          = "soundboard-wireguard"
  network       = "default"
  target_tags   = ["soundboard"]
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "udp"
    ports    = ["51820"]
  }
  depends_on = [google_project_service.services]
}

resource "google_compute_firewall" "ssh" {
  name          = "soundboard-ssh"
  network       = "default"
  target_tags   = ["soundboard"]
  source_ranges = var.ssh_source_ranges

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
  depends_on = [google_project_service.services]
}

# ---- Imágenes Docker ----

# Desde la Mac se hace push (solo suben las capas que cambiaron) y la VM hace pull por la red de
# Google. El free tier cubre 0.5 GB; la limpieza borra las versiones que pierden el tag `latest`.
resource "google_artifact_registry_repository" "images" {
  repository_id = "soundboard"
  location      = var.gcp_region
  format        = "DOCKER"
  depends_on    = [google_project_service.services]

  cleanup_policies {
    id     = "borrar-sin-tag"
    action = "DELETE"
    condition {
      tag_state  = "UNTAGGED"
      older_than = "86400s"
    }
  }
}

resource "google_artifact_registry_repository_iam_member" "images_vm" {
  location   = google_artifact_registry_repository.images.location
  repository = google_artifact_registry_repository.images.name
  role       = "roles/artifactregistry.reader"
  member     = google_service_account.vm.member
}

# ---- Medios (sonidos, imágenes y uploads temporales) ----

resource "google_storage_bucket" "media" {
  name                        = "soundboard-media-${random_id.suffix.hex}"
  location                    = upper(var.gcp_region)
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  depends_on                  = [google_project_service.services]

  cors {
    origin          = [local.admin_origin]
    method          = ["GET", "HEAD"]
    response_header = ["Content-Type", "Cache-Control", "ETag"]
    max_age_seconds = 3600
  }

  # La API borra uploads/ cuando el job termina bien; esto limpia los de jobs fallidos.
  lifecycle_rule {
    condition {
      age            = 1
      matches_prefix = ["uploads/"]
    }
    action {
      type = "Delete"
    }
  }
}

# Lectura pública por key sin permiso de listar (legacyObjectReader no incluye objects.list).
resource "google_storage_bucket_iam_member" "media_public" {
  bucket = google_storage_bucket.media.name
  role   = "roles/storage.legacyObjectReader"
  member = "allUsers"
}

# La API corre en la VM y usa su cuenta de servicio (metadata server): sin claves que guardar.
resource "google_storage_bucket_iam_member" "media_vm" {
  bucket = google_storage_bucket.media.name
  role   = "roles/storage.objectAdmin"
  member = google_service_account.vm.member
}

# ---- Respaldos de la base (SQLite de D1) ----

resource "google_storage_bucket" "backups" {
  name                        = "soundboard-backups-${random_id.suffix.hex}"
  location                    = upper(var.gcp_region)
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  depends_on                  = [google_project_service.services]

  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type = "Delete"
    }
  }
}

resource "google_storage_bucket_iam_member" "backups_vm" {
  bucket = google_storage_bucket.backups.name
  role   = "roles/storage.objectCreator"
  member = google_service_account.vm.member
}
