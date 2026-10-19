import errno,json,os,sys
result={}
for name,flags in [('read',os.O_RDONLY),('write_without_truncate',os.O_WRONLY)]:
 try:
  fd=os.open(sys.argv[1],flags);os.close(fd);result[name]={'denied':False}
 except OSError as e:result[name]={'denied':e.errno in (errno.EPERM,errno.EACCES),'errno':e.errno}
print(json.dumps(result))
