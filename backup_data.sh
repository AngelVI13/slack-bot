#!/usr/bin/bash
eval $(keychain --eval /home/angel/.ssh/id_ed25519)

timestamp() {
    date +"%Y-%m-%d_%H-%M-%S"
}
file_prefix=$(timestamp)
backup_dir="./backups/${file_prefix}"
mkdir -p $backup_dir

remote_dir="/home/tmt/Documents/slack/slack-bot"

ssh tmt@172.20.2.200 'sudo journalctl -u parking_bot.service' > "${backup_dir}/parking.log"
scp tmt@172.20.2.200:$remote_dir/parking.json "${backup_dir}/parking.json"
scp tmt@172.20.2.200:$remote_dir/workspaces.json "${backup_dir}/workspaces.json"
scp tmt@172.20.2.200:$remote_dir/users.json "${backup_dir}/users.json"
scp tmt@172.20.2.200:$remote_dir/vacations_hash.json "${backup_dir}/vacations_hash.json"
scp tmt@172.20.2.200:$remote_dir/slack-bot.log "${backup_dir}/slack-bot.log"
scp tmt@172.20.2.200:$remote_dir/slack-bot "${backup_dir}/slack-bot"

# Remove some noise from the log
sed -i 's/tmt-pro.*level=//' "${backup_dir}/parking.log"

