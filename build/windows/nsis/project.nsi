Unicode true

!define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
!define SYNC_MANAGER_EXECUTABLE "sync-manager.exe" # Add this line
!define REQUEST_EXECUTION_LEVEL "admin"

SetCompressor /SOLID lzma
SetCompress auto

!include "wails_tools.nsh"
!include "LogicLib.nsh"
!include "WinVer.nsh"

!include "MUI2.nsh"
!include "WinMessages.nsh"

# The version information
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support
ManifestDPIAware true
ManifestSupportedOS all

!include "MUI2.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING

# Variable to track if app was running before install
Var WasAppRunning

# Define finish page behavior
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "Launch ${INFO_PRODUCTNAME}"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-Setup.exe"
InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
ShowInstDetails show

# Function to check if service exists and stop/uninstall it if needed
Function CheckAndUninstallService
    DetailPrint "Checking for existing Sync Manager service..."

    # Check if the service exists by trying to query its status
    nsExec::ExecToStack 'sc query "jarvist-sync"'
    Pop $0
    Pop $1

    # If service exists, proceed with stopping and uninstalling
    ${If} $0 == "0"
        DetailPrint "Sync Manager service found. Stopping service..."

        # First try to stop the service
        nsExec::ExecToStack 'sc stop "jarvist-sync"'
        Pop $0
        Pop $1

        # Wait a moment for the service to stop
        Sleep 2000

        # Uninstall the service
        DetailPrint "Uninstalling existing Sync Manager service..."

        # Check if sync-manager.exe exists in the current installation
        IfFileExists "$INSTDIR\${SYNC_MANAGER_EXECUTABLE}" 0 +3
            DetailPrint "Using existing sync-manager.exe to uninstall service..."
            nsExec::ExecToStack '"$INSTDIR\${SYNC_MANAGER_EXECUTABLE}" --uninstall'
            Goto ServiceUninstallAttempted

        # Check if sync-manager.exe exists in the temp directory (newly extracted)
        IfFileExists "$TEMP\${SYNC_MANAGER_EXECUTABLE}" 0 +3
            DetailPrint "Using temp sync-manager.exe to uninstall service..."
            nsExec::ExecToStack '"$TEMP\${SYNC_MANAGER_EXECUTABLE}" --uninstall'
            Goto ServiceUninstallAttempted

        # Fallback to the default Windows sc command
        DetailPrint "Using sc command to remove service..."
        nsExec::ExecToStack 'sc delete "jarvist-sync"'

        ServiceUninstallAttempted:
        # Wait for the service to be completely removed
        Sleep 2000

        # Verify service is gone
        nsExec::ExecToStack 'sc query "jarvist-sync"'
        Pop $0
        Pop $1

        ${If} $0 == "0"
            DetailPrint "Warning: Service may still be registered. Will continue anyway..."
        ${Else}
            DetailPrint "Service successfully uninstalled."
        ${EndIf}
    ${Else}
        DetailPrint "No existing Sync Manager service found."
    ${EndIf}
FunctionEnd

# Function to detect and close running application
Function CloseRunningApp
    StrCpy $WasAppRunning "0"

    nsExec::ExecToStack 'tasklist /FI "IMAGENAME eq ${PRODUCT_EXECUTABLE}" /NH'
    Pop $0
    Pop $1

    ${If} $1 != ""
        ${If} $1 != "INFO: No tasks are running which match the specified criteria."
            StrCpy $WasAppRunning "1"

            DetailPrint "Attempting to close ${INFO_PRODUCTNAME}..."
            nsExec::ExecToStack 'taskkill /IM ${PRODUCT_EXECUTABLE}'
            Pop $0
            Pop $1
            Sleep 1000

            nsExec::ExecToStack 'tasklist /FI "IMAGENAME eq ${PRODUCT_EXECUTABLE}" /NH'
            Pop $0
            Pop $1

            ${If} $1 != "INFO: No tasks are running which match the specified criteria."
                DetailPrint "Force closing ${INFO_PRODUCTNAME}..."
                nsExec::ExecToStack 'taskkill /F /IM ${PRODUCT_EXECUTABLE}'
                Pop $0
                Pop $1
                Sleep 500
            ${EndIf}
        ${EndIf}
    ${EndIf}
FunctionEnd

Function .onInit
    !insertmacro wails.checkArchitecture

    ${IfNot} ${AtLeastWin7}
        MessageBox MB_ICONSTOP "This application requires Windows 7 or later."
        Abort
    ${EndIf}

    Call CloseRunningApp

    DetailPrint "Stopping all FFmpeg processes..."
    nsExec::ExecToStack 'taskkill /F /IM ffmpeg.exe'
    Pop $0
    Pop $1
    Sleep 1000
FunctionEnd

Section
    DetailPrint "Checking and uninstalling existing Sync Manager service..."
    Call CheckAndUninstallService

    DetailPrint "Ensuring application is not running..."
    nsExec::ExecToStack 'taskkill /F /IM ${PRODUCT_EXECUTABLE}'
    Pop $0
    Pop $1

    DetailPrint "Stopping all FFmpeg processes..."
    nsExec::ExecToStack 'taskkill /F /IM ffmpeg.exe'
    Pop $0
    Pop $1
    Sleep 1000

    !insertmacro wails.setShellContext
    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR
    !insertmacro wails.files

    # Extract bin directory using 7-Zip
    DetailPrint "Preparing to extract binary files..."
    SetOutPath $INSTDIR
    File "7za.exe"
    File "bin.7z"
    File "..\..\..\bin\sync-manager.exe"

    SetDetailsPrint both
    DetailPrint "Extracting binary files (this may take several minutes)..."
    SetDetailsPrint listonly

    nsExec::ExecToStack '"$INSTDIR\7za.exe" x "$INSTDIR\bin.7z" -o"$INSTDIR" -y'
    Pop $0
    Pop $1

    SetDetailsPrint both

    ${If} $0 != "0"
        MessageBox MB_ICONSTOP "Extraction failed! Error code: $0"
        Abort
    ${EndIf}

    # Clean up extraction tools
    Delete "$INSTDIR\7za.exe"
    Delete "$INSTDIR\bin.7z"

    # Verify extraction succeeded
    DetailPrint "Verifying installation..."
    IfFileExists "$INSTDIR\bin\mingw64\gcc.exe" GccExists
        MessageBox MB_ICONSTOP "GCC installation failed! Required files not found."
        Abort
    GccExists:

    IfFileExists "$INSTDIR\bin\ffmpeg\ffmpeg.exe" FfmpegExists
        MessageBox MB_ICONSTOP "FFmpeg installation failed! Required files not found."
        Abort
    FfmpegExists:

    IfFileExists "$INSTDIR\sync-manager.exe" SyncManagerExists
        MessageBox MB_ICONSTOP "Sync Manager installation failed! Required file not found."
        Abort
    SyncManagerExists:

    # Create additional directories
    CreateDirectory "$INSTDIR\logs"

    # Update system PATH
    DetailPrint "Updating system PATH..."
    EnVar::SetHKCU
    EnVar::AddValue "PATH" "$INSTDIR\bin\mingw64"
    EnVar::AddValue "PATH" "$INSTDIR\bin\ffmpeg"

    # Create shortcuts
    DetailPrint "Creating shortcuts..."
    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.writeUninstaller

    # Handle application launch after installation
    DetailPrint "Finalizing installation..."
    ${If} $WasAppRunning == "1"
        DetailPrint "The application was running before installation. Restarting..."
        Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}"'
    ${EndIf}
SectionEnd

Section "uninstall"
    DetailPrint "Closing application if running..."
    nsExec::ExecToStack 'taskkill /F /IM ${PRODUCT_EXECUTABLE}'
    Pop $0
    Pop $1
    Sleep 1000

    DetailPrint "Checking for running Sync Manager service..."
    nsExec::ExecToStack 'sc query "jarvist-sync"'
    Pop $0
    Pop $1

    ${If} $0 == "0"
        DetailPrint "Stopping Sync Manager service..."
        nsExec::ExecToStack 'sc stop "jarvist-sync"'
        Pop $0
        Pop $1
        Sleep 2000

        DetailPrint "Uninstalling Sync Manager service..."
        nsExec::ExecToStack '"$INSTDIR\${SYNC_MANAGER_EXECUTABLE}" --uninstall'
        Pop $0
        Pop $1

        # If the uninstall via sync-manager.exe fails, try using sc delete as a fallback
        ${If} $0 != "0"
            DetailPrint "Using sc command to remove service..."
            nsExec::ExecToStack 'sc delete "jarvist-sync"'
            Pop $0
            Pop $1
        ${EndIf}

        Sleep 1000
    ${EndIf}

    DetailPrint "Stopping all FFmpeg processes..."
    nsExec::ExecToStack 'taskkill /F /IM ffmpeg.exe'
    Pop $0
    Pop $1
    Sleep 1000

    !insertmacro wails.setShellContext

    DetailPrint "Removing application data..."
    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}"
    RMDir /r "$INSTDIR\logs"
    RMDir /r $INSTDIR

    DetailPrint "Removing shortcuts..."
    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    DetailPrint "Updating system PATH..."
    EnVar::SetHKCU
    EnVar::DeleteValue "PATH" "$INSTDIR\bin\mingw64"
    EnVar::DeleteValue "PATH" "$INSTDIR\bin\ffmpeg"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.deleteUninstaller

    DetailPrint "Uninstallation complete."
SectionEnd
