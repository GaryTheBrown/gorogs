## FIXES:
1. windows samba share in kodi not populating wsdd fully compliant so not sure what else is wrong need to look into this more
2. nfs share in kodi not working at all (freezes then fails to enter) could this be cause of the udp? maybe make a helper that works on the host thatcaptures the udp traffic and passes it to the individual servers or it grabs the info, updates when they report and it then tells the net on request about available shares??
3. nfs is empty in dolphin.
   
## ADDS:
1. Hot Reload of samba config for dynamic shares (folders in  /srv that must act like shares)
2. make all env vars comand options that if set take priority so you can build a version with nfs out and it will never be allowed to run from entrypoint or even have enabled options so even trying to disable the option in the compose will not disable it.
3. a better disable/enable env var setup where user can list items. make it so each beacon/share (plugin) is individually added to the list like debul_log . this will also include adding in better handeling of the system. try and detach the config and have the relavent data pass in to the (plugin) on setup this needs to be generic enough for it to be completly seperated before.
4. need to improve the plugins so that they say what they require and and if one must start before the other.
5. use an overlayfs to make the shares show as full and not the real remaining size so even if they somehow get it into a writable state they still cannot.
 