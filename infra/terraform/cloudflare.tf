data "cloudflare_zone" "main" {
  filter = {
    name    = var.cloudflare_zone
    account = var.cloudflare_account_id == null ? null : { id = var.cloudflare_account_id }
  }
}

# Sin proxy (nube gris): Universal SSL no cubre nombres de 4.º nivel, así que el certificado lo
# sirve nginx en la VM (wildcard de Let's Encrypt emitido con DNS-01).
resource "cloudflare_dns_record" "app" {
  for_each = toset(["api", "admin"])

  zone_id = data.cloudflare_zone.main.id
  name    = "${each.key}.${var.base_domain}"
  type    = "A"
  content = google_compute_address.soundboard.address
  proxied = false
  ttl     = 300
  comment = "Soundboard (Terraform)"
}
