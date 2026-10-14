variable "gcp_project" {
  description = "Proyecto de GCP donde vive la VM y los buckets."
  type        = string
  default     = "zenbash"
}

# El free tier de e2-micro y de Cloud Storage solo aplica en us-central1, us-east1 y us-west1.
variable "gcp_region" {
  type    = string
  default = "us-central1"
}

variable "gcp_zone" {
  type    = string
  default = "us-central1-a"
}

variable "cloudflare_api_token" {
  description = "Token de Cloudflare con Zone:DNS:Edit. Si es null se lee CLOUDFLARE_API_TOKEN."
  type        = string
  default     = null
  sensitive   = true
}

variable "cloudflare_account_id" {
  description = "Cuenta de Cloudflare (opcional). Si se define, la zona se busca solo en esa cuenta."
  type        = string
  default     = null
}

variable "cloudflare_zone" {
  description = "Zona de Cloudflare que contiene los registros."
  type        = string
  default     = "suave.sh"
}

variable "base_domain" {
  description = "Dominio bajo el que se publican api. y admin."
  type        = string
  default     = "soundboard.homunculos.suave.sh"
}

variable "ssh_user" {
  description = "Usuario con el que Ansible entra a la VM."
  type        = string
  default     = "alchemist"
}

variable "ssh_public_key_path" {
  type    = string
  default = "~/.ssh/id_ed25519.pub"
}

variable "ssh_source_ranges" {
  description = "CIDRs que pueden entrar por SSH (p. ej. tu IP pública /32)."
  type        = list(string)
}
