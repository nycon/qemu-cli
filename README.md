# qemu-cli

Official Go command-line interface for [QEMU Interface](https://github.com/nycon/qemu-wi) panels.

Manage VMs, snapshots, backups, tenants, and security from the terminal.

## Installation

```bash
# Go 1.22+
git clone https://github.com/nycon/qemu-cli.git
cd qemu-cli
make install
```

Or:

```bash
go install github.com/nycon/qemu-cli@latest
```

Ensure `$(go env GOPATH)/bin` (and optionally `~/.local/bin`) is on your `PATH`.

## Getting Started

1. Create an API token in the panel: **Settings > API Keys > New API Key**
2. Create a CLI context:

```bash
qemu-cli context create my-server \
  --url https://panel.example.com \
  --token qi_your_api_key
```

For self-signed TLS certificates add `--insecure`.

3. Start managing VMs:

```bash
qemu-cli vm list
qemu-cli vm start my-webserver
```

## Commands

```
qemu-cli context create <name>     Create a new server context
qemu-cli context list              List all contexts
qemu-cli context use <name>        Switch active context
qemu-cli context delete <name>     Remove a context
qemu-cli context active            Show active context

qemu-cli vm list                   List all VMs
qemu-cli vm create <name>          Create a new VM
qemu-cli vm describe <id|name>     Show VM details
qemu-cli vm start <id|name>        Start a VM
qemu-cli vm stop <id|name>         Gracefully stop a VM
qemu-cli vm reboot <id|name>       Reboot a VM
qemu-cli vm poweroff <id|name>     Force power-off a VM
qemu-cli vm delete <id|name>       Delete a VM
qemu-cli vm resize <id|name>       Resize CPU/RAM/Disk

qemu-cli snapshot list <vm>        List snapshots
qemu-cli snapshot create <vm>      Create a snapshot
qemu-cli snapshot restore <vm> <n> Restore a snapshot
qemu-cli snapshot delete <vm> <n>  Delete a snapshot

qemu-cli backup list <vm>          List backups
qemu-cli backup create <vm>        Create a backup
qemu-cli backup restore <vm> <id>  Restore from backup
qemu-cli backup delete <vm> <id>   Delete a backup

qemu-cli storage-pool list         List storage pools
qemu-cli image list                List cloud images

qemu-cli firewall list             List banned IPs
qemu-cli firewall ban <ip>         Ban an IP
qemu-cli firewall unban <ip>       Unban an IP
qemu-cli firewall check <ip>       Check if IP is banned
qemu-cli firewall jails            Show Fail2Ban jail config
qemu-cli firewall set-jail <jail>  Update jail settings
qemu-cli firewall log              View Fail2Ban log

qemu-cli tenant list               List tenants
qemu-cli tenant use <id|name>      Select active tenant
qemu-cli user list                 List users

qemu-cli server-info               Show host CPU/RAM/Disk
qemu-cli version                   Show CLI version
qemu-cli completion zsh            Shell completion script
```

## Output Formats

```bash
qemu-cli vm list
qemu-cli -o json vm list
qemu-cli -o yaml vm describe my-webserver
```

## Multiple Servers

```bash
qemu-cli context create production --url https://panel.example.com --token qiqi_...
qemu-cli context create staging --url https://staging.example.com --token qiqi_...

qemu-cli context use production
qemu-cli vm list
```

## Partial IDs and Names

```bash
qemu-cli vm start 8b859dca-b3d5-4f8a-8a19-ae403426073e
qemu-cli vm start 8b859
qemu-cli vm start my-webserver
```

## Creating VMs

```bash
qemu-cli vm create my-webserver \
  --image "Ubuntu 24.04" \
  --project <project-id> \
  --ip-pool <ip-pool-id> \
  --random-password

qemu-cli vm create my-webserver \
  --image "Ubuntu 24.04" \
  --project <project-id> \
  --ip-pool <ip-pool-id> \
  --cpu 2 --ram 2048 --disk 30 \
  --password "MySecurePassword123!" \
  --ssh-key "ssh-ed25519 AAAA... user@host"
```

### VM Create Options

| Option | Default | Description |
|---|---|---|
| `--image` | (required) | Cloud image ID or name |
| `--project` | (required) | Project UUID |
| `--ip-pool` | (required) | IP pool UUID |
| `--cpu` | 1 | Number of vCPUs |
| `--ram` | 1024 | RAM in MB |
| `--disk` | 20 | Disk size in GB |
| `--password` | - | Guest password |
| `--random-password` | - | Generate a 20-char random password |
| `--ssh-key` | - | SSH public key |
| `--hostname` | - | Custom hostname |
| `--user` | cloud | Default cloud-init user |
| `--storage-pool` | - | Storage pool UUID |

## Tenant isolation

API keys authenticate as a panel user. Non-superadmin users only see their tenants; VM actions are scoped to `/tenants/{id}/...` membership checks.

## Configuration

Stored in `~/.qemu-cli/config.json` (mode `0600`):

```json
{
  "active_context": "production",
  "contexts": {
    "production": {
      "server_url": "https://panel.example.com",
      "token": "qi_abc123...",
      "tenant_id": "...",
      "insecure": false
    }
  }
}
```

## Shell Completion

```bash
qemu-cli completion zsh > "${fpath[1]}/_qemu-cli"
qemu-cli completion bash > /etc/bash_completion.d/qemu-cli
qemu-cli completion fish > ~/.config/fish/completions/qemu-cli.fish
```

## Related

- Panel: https://github.com/nycon/qemu-wi

## License

MIT
