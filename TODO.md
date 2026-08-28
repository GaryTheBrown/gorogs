## TODO FIXES:
1. windows samba share in kodi not populating it's hostname but only it's ip (connected to old netbios stuff in kodi so it needs talking about with them)
2. nfs share in kodi not working at all (freezes then fails to enter) could this be cause of the udp? maybe make a helper that works on the host that captures the udp traffic and passes it to the individual servers or it grabs the info, updates when they report and it then tells the net on request about available shares??
3. nfs is empty in dolphin.
   
## TODO ADDS:
1. Need to go through all the programs we call and look at what logging they each do and have it give the basic and change to debugging when debugging is active. need to also look at what it outputs and and make it do some magic on the data to strip away data we provide and if possible tell what log level it should use.
2. be able to say a system requires another system so we can do some ordering of startups.
