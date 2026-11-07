$ErrorActionPreference = 'Stop'
# Native runtime build. Uses an ASCII working copy (C:\rmm_native\Source) to
# keep the build hermetic around Windows codepages; that copy is refreshed
# from this repository on every run.
$repo = Split-Path -Parent $MyInvocation.MyCommand.Path
$mirrorRoot = 'C:\rmm_native'
$mirrorSource = Join-Path $mirrorRoot 'Source'

if (Test-Path $mirrorSource) { Remove-Item -Recurse -Force $mirrorSource }
New-Item -ItemType Directory -Force $mirrorRoot | Out-Null
Copy-Item -Recurse $repo $mirrorSource

$BuildRoot = Join-Path $mirrorSource 'build'
$CMakeBuild = Join-Path $BuildRoot 'native-build'
$LLVM = Join-Path $env:ProgramFiles 'LLVM'
$ClangCl = Join-Path $LLVM 'bin\clang-cl.exe'
$env:Path = "$LLVM\bin;C:\Program Files\Go\bin;C:\Program Files\CMake\bin;$env:Path"
$Version = (Get-Content (Join-Path $repo 'VERSION') -TotalCount 1).Trim()
if (-not $Version) { $Version = '1.0.0' }

$DevCmd = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio\2022\BuildTools\Common7\Tools\VsDevCmd.bat'
$runner = Join-Path $BuildRoot 'run_native.cmd'
$scriptBody = @"
call "$DevCmd" -arch=amd64 -host_arch=amd64
if exist "$CMakeBuild" rmdir /s /q "$CMakeBuild"
cmake -S $mirrorSource -B $CMakeBuild -G Ninja -DCMAKE_CXX_COMPILER="$ClangCl" -DCMAKE_BUILD_TYPE=Release -DCMAKE_SHARED_LINKER_FLAGS="msvcrt.lib vcruntime.lib ucrt.lib"
cmake --build $CMakeBuild
"@
if (-not (Test-Path $BuildRoot)) { New-Item -ItemType Directory -Force $BuildRoot | Out-Null }
Set-Content -Path $runner -Value $scriptBody -Encoding Oem

cmd /c $runner
if ($LASTEXITCODE -ne 0) {
    throw 'CMake build of the native runtime failed.'
}

$Dll = Get-ChildItem -Path $CMakeBuild -Filter 'rml_runtime.dll' -File -Recurse |
    Sort-Object LastWriteTime -Descending |
    Select-Object -First 1
if (-not $Dll) {
    throw 'CMake completed but rml_runtime.dll was not found.'
}
$RepoBuild = Join-Path $repo 'build'
New-Item -ItemType Directory -Force $RepoBuild | Out-Null
$Out = Join-Path $RepoBuild 'rml_runtime.dll'
Copy-Item -Force $Dll.FullName $Out
python (Join-Path $repo 'tools\pe_resources.py') $Out `
    --version $Version `
    --file-description 'Republic Mod manager Runtime' `
    --original-filename 'rml_runtime.dll' `
    --dll
python (Join-Path $Root 'tools\pe_finalize.py') $Out
Write-Host "Built Republic Mod manager Runtime v$Version at $Out"
