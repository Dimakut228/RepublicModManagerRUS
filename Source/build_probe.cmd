call "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\Common7\Tools\VsDevCmd.bat" -arch=amd64 -host_arch=amd64
echo LIB=[%LIB%]
dir /b "%LIB:;=" & dir /b "%" 2>NUL
where lld-link.exe
where ntdll.lib
echo DONE
