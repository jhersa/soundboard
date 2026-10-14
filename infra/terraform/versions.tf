terraform {
  required_version = ">= 1.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

# Credenciales:
#   gcloud auth application-default login
#   Cloudflare: `cloudflare_api_token` en terraform.tfvars (ignorado en git) o
#   export CLOUDFLARE_API_TOKEN=...   (Zone:DNS:Edit sobre la zona)
provider "google" {
  project = var.gcp_project
  region  = var.gcp_region
  zone    = var.gcp_zone
}

# Si la variable es null, el provider usa CLOUDFLARE_API_TOKEN del entorno.
provider "cloudflare" {
  api_token = var.cloudflare_api_token
}
