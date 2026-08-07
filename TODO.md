## FIXES:
1. nfs not booting.
2. zeroconf using .local and not FQDN
3. windows samba share in kodi not populating
4. nfs share in kodi not working at all (freezes then fails to enter)

## ADDS:
1. make all env vars comand options that if set take priority so you can build a version with nfs out and it will never be allowed to run from entrypoint
2. can we take logs from the underlying programs running, capture them and convert them to our logging
3. loose the app folder
4. if debug has rpcbind, nfs or samba systems (or any systems we didn't write) in can we make the system output more logs (maximum ammount possible for each app)
