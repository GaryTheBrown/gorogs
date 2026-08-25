## TODO FIXES:
1. windows samba share in kodi not populating it's hostname but only it's ip (connected to old netbios stuff in kodi so it needs talking about with them)
2. nfs share in kodi not working at all (freezes then fails to enter) could this be cause of the udp? maybe make a helper that works on the host that captures the udp traffic and passes it to the individual servers or it grabs the info, updates when they report and it then tells the net on request about available shares??
3. nfs is empty in dolphin.
   
## TODO ADDS:
1. be able to say a system requires another system so we can do some ordering of startups.
