@echo off
setlocal enabledelayedexpansion

set "package_name=FileCleanup"
set "build_folder="

rem Define the list of target platforms
rem add linux env linux/386 linux/amd64 linux/arm
set "platforms=windows/amd64 windows/386"

rem Loop through each platform
for %%P in (%platforms%) do (
    for /f "tokens=1,2 delims=/" %%A in ("%%P") do (
        set "GOOS=%%A"
        set "GOARCH=%%B"

        rem Determine the build folder based on the platform
        if "%%A" == "windows" (
            set "build_folder=windows-build"
        ) else (
            set "build_folder=linux-build"
        )

        rem Create the build folder if it doesn't exist
        if not exist "!build_folder!" (
            mkdir "!build_folder!"
        )


        rem Build the output name based on the platform
        set "output_name=!package_name!-!GOOS!-!GOARCH!"
        if "!GOOS!" == "windows" set "output_name=!output_name!.exe"

        rem Build the executable
        set "command=go build -o !output_name! main.go"
        if "!GOOS!" == "windows" set "command=go build -o !output_name! main.go"
        call !command!

        rem Move the executable to the build folder
        move "!output_name!" "!build_folder!\!output_name!"

        rem Check for errors during build
        if errorlevel 1 (
            echo An error has occurred! Aborting the script execution...
            exit /b 1
        )
    )
)

rem Zip the build folders
set "zip_name=FileCleanup.zip"
rem powershell Compress-Archive -Path "windows-build", "linux-build" -DestinationPath "!zip_name!"
powershell Compress-Archive -Path "windows-build" -DestinationPath "!zip_name!"

echo Builds have been zipped to !zip_name!

endlocal
