## TODO UPDATES AND IMPORTANTS
1. Review all logger calls and make a lot of them debug calls keeping minimal showing the systems loading and thats about it the rest of the logger calls should be debug only fatal is left alone
2. fix the way the config section works and put as much info in the fixed variables and make sure anywhere that wants them grabs them from the config section.
3. expand in a way so each share and beacon have there own configs that we grab from the env using gorogs.global.[variable] and gorogs.[system].[variable] (or deeper if the config is needing more complex configuration).
## TODO FIXES:
1. windows samba share in kodi not populating it's hostname but only it's ip (connected to old netbios stuff in kodi so it needs talking about with them)
2. nfs share in kodi not working at all (freezes then fails to enter) could this be cause of the udp? maybe make a helper that works on the host thatcaptures the udp traffic and passes it to the individual servers or it grabs the info, updates when they report and it then tells the net on request about available shares??
3. nfs is empty in dolphin.
   
## TODO ADDS:
1. simplify all systems struct file and clean up all the logger.info to logger.debug if it's the struct itself
 

## TODO MAYBE?
1. make all env vars comand options that if set take priority so you can build a version with nfs out and it will never be allowed to run from entrypoint or even have enabled options so even trying to disable the option in the compose will not disable it.