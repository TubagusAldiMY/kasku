# Cloudflare Tunnel — Local Development

Panduan menjalankan Cloudflare Tunnel dari mesin lokal (dev machine), termasuk fix konflik dengan Cloudflare WARP.

---

## Prasyarat

- `cloudflared` sudah terinstall
- Token tunnel sudah dibuat di Cloudflare Zero Trust dashboard
- (Opsional) Cloudflare WARP aktif → baca seksi [Konflik WARP](#konflik-cloudflare-warp) dulu

---

## 1. Install cloudflared

```bash
# Debian / Ubuntu
curl -L --output cloudflared.deb \
  https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb

# Verifikasi
cloudflared --version
```

---

## 2. Jalankan Tunnel

### Mode Cepat (sekali jalan)

```bash
cloudflared tunnel run --token <YOUR_TUNNEL_TOKEN>
```

### Mode Recommended (HTTP/2 eksplisit, lebih stabil di jaringan yang memblokir UDP)

```bash
cloudflared tunnel run \
  --token <YOUR_TUNNEL_TOKEN> \
  --protocol http2
```

### Jika jaringan hanya support IPv6

```bash
cloudflared tunnel run \
  --token <YOUR_TUNNEL_TOKEN> \
  --protocol http2 \
  --edge-ip-version 6
```

---

## 3. Konfigurasi Permanen (config file)

Buat file `~/.cloudflared/config.yml` agar tidak perlu flag setiap kali:

```yaml
# ~/.cloudflared/config.yml
tunnel: <TUNNEL_ID>
credentials-file: /home/<user>/.cloudflared/<TUNNEL_ID>.json

# Ingress rules
ingress:
  - hostname: api-kasku.tubsamy.dev
    service: http://localhost:18080
  - service: http_status:404

# Network settings (stabil di jaringan yang blokir UDP)
protocol: http2
```

Jalankan cukup dengan:

```bash
cloudflared tunnel run
```

---

## 4. Verifikasi Koneksi

Cek log untuk baris ini — artinya tunnel berhasil konek:

```
INF Registered tunnel connection connIndex=0 ... location=sin08 protocol=http2
INF Updated to new configuration config="{\"ingress\":[...]}"
```

Tunnel berjalan normal dengan **minimal 1 koneksi aktif**. Normalnya cloudflared membuka 4 koneksi paralel untuk redundansi — jika hanya 1-2 yang terbuka, tunnel tetap fungsional tapi kurang redundant.

---

## Konflik: Cloudflare WARP

### Mengapa Konflik Terjadi

WARP menggunakan WireGuard untuk merouting **semua** traffic melalui Cloudflare network. Ketika cloudflared mencoba konek ke Cloudflare edge di port 7844, traffic tersebut disita WARP dan dirouting ulang — menyebabkan loop/deadlock koneksi.

Gejala yang muncul di log:

```
FAIL  QUIC connection failed
FAIL  HTTP/2 connection is blocked or unreachable
ERROR: Allow outbound QUIC traffic on port 7844 or use HTTP2.
ERROR: Allow outbound TCP on port 7844.
WRN   Unable to establish connection ... i/o timeout
```

### Solusi: WARP Split Tunnel Exclusion

Tambahkan IP range Cloudflare argotunnel ke exclude list WARP, sehingga cloudflared konek **langsung** ke edge tanpa lewat WARP:

```bash
# IPv4 argotunnel range (mencakup 198.41.192.x dan 198.41.200.x)
warp-cli split-tunnel exclude add 198.41.192.0/20

# IPv6 argotunnel ranges
warp-cli split-tunnel exclude add 2606:4700:a0::/48
warp-cli split-tunnel exclude add 2606:4700:a8::/48
```

Reconnect WARP setelah menambahkan exclusion:

```bash
warp-cli disconnect && warp-cli connect
```

Verifikasi exclusion sudah ditambahkan:

```bash
warp-cli split-tunnel exclude list
```

Jalankan ulang cloudflared — tunnel seharusnya konek penuh (4 koneksi):

```bash
cloudflared tunnel run --token <YOUR_TUNNEL_TOKEN> --protocol http2
```

### Via WARP Desktop App (GUI)

**Settings → Network → Split Tunneling → Exclude IPs** → tambahkan:

| CIDR | Keterangan |
|------|-----------|
| `198.41.192.0/20` | Cloudflare argotunnel IPv4 |
| `2606:4700:a0::/48` | Cloudflare argotunnel IPv6 region1 |
| `2606:4700:a8::/48` | Cloudflare argotunnel IPv6 region2 |

---

## Troubleshooting

### Hanya 1 dari 4 koneksi yang berhasil

Jaringan memblokir IPv4 port 7844 tapi IPv6 lolos. Tunnel tetap fungsional. Fix dengan exclusion WARP di atas, atau paksa IPv6:

```bash
cloudflared tunnel run --token <TOKEN> --protocol http2 --edge-ip-version 6
```

### `ping_group_range` warning

```
WRN Group ID 1000 is not between ping group 1 to 0
```

Warning ini aman diabaikan — hanya menonaktifkan fitur ICMP proxy, tidak mempengaruhi tunnel HTTP.

### UDP buffer size warning

```
failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 7168 kiB, got: 416 kiB)
```

Opsional, untuk performa QUIC lebih baik:

```bash
sudo sysctl -w net.core.rmem_max=7340032
sudo sysctl -w net.core.wmem_max=7340032
```

Permanen (tambahkan ke `/etc/sysctl.conf`):

```
net.core.rmem_max=7340032
net.core.wmem_max=7340032
```

### Cek connectivity manual

```bash
# Test TCP ke argotunnel (harus bisa konek)
nc -zv region1.v2.argotunnel.com 7844

# Cek DNS resolve
dig region1.v2.argotunnel.com
dig region2.v2.argotunnel.com
```

---

## Referensi

- Tunnel ini dikonfigurasi via Cloudflare Zero Trust Dashboard → Networks → Tunnels
- Ingress rules dikelola dari dashboard (bukan config file) jika menggunakan remote config
- Untuk setup VPS production, lihat [`VPS_SETUP.md`](./VPS_SETUP.md)
