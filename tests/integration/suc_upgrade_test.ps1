$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

# clean interferences
try {
    Get-Process -Name "rancher-wins-*" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
    Get-NetFirewallRule -PolicyStore ActiveStore -Name "rancher-wins-*" -ErrorAction Ignore | ForEach-Object { Remove-NetFirewallRule -Name $_.Name -PolicyStore ActiveStore -ErrorAction Ignore } | Out-Null
    Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
}
catch {
    Log-Warn $_.Exception.Message
}

Describe "install" {
    BeforeEach {
        Log-Info "Running install script"
        # note: Simply running the install script does not do anything. During normal provisioning,
        # Rancher will mutate the install script to both add environment variables, and to call
        # the primary function 'Invoke-WinsInstaller'. As this is an integration test, we need to manually
        # update the install script ourselves.
        Add-Content -Path ./install.ps1 -Value '$env:CATTLE_REMOTE_ENABLED = "false"'
        Add-Content -Path ./install.ps1 -Value '$env:CATTLE_LOCAL_ENABLED = "true"'
        Add-Content -Path ./install.ps1 -Value Invoke-WinsInstaller

        .\install.ps1
    }

    AfterEach {
        Log-Info "Running uninstall script"
        try
        {
            # note: since this script may not be run by an administrator, it's possible that it might fail
            # on trying to delete certain files with ACLs attached to them.
            # If you are running this locally, make sure you run with admin privileges.
            # On CI, since we don't run as an admin today, this prevents automatic failure when the right ACLs are set.
            .\uninstall.ps1
        }
        catch
        {
            Log-Warn "You need to manually run uninstall.ps1, encountered error: $( $_.Exception.Message )"
        }
    }

    It "Installs and upgrades" {
        # We currently have the latest release installed, we now need to test upgrading to our version.
        # Get the expected version of the new wins.exe binary. On PR's this
        # will be a commit hash, and on tag runs it should be a full version (v0.x.y[-rc.z]).
        $currentCommit = Get-LatestCommitOrTag
        Log-Info "Expected wins.exe version: $currentCommit"

        # Get the currently installed version string
        $fullVersion = $(c:\Windows\wins.exe --version)
        Log-Info $fullVersion
        $initialVersion = $(c:\Windows\wins.exe --version).Split(" ")[2]
        $initialVersion -eq "" | Should -BeFalse
        Log-Info "Initial wins.exe version is $initialVersion"

        # Run the suc image manually
        Log-Info "Executing wins-suc.exe"
        $env:CATTLE_WINS_UPGRADE_BINARY="true"
        $env:CATTLE_WINS_DEBUG="true"
        Execute-Binary -FilePath "bin\wins-suc.exe"
        Log-Info "Command exited successfully: $?"
        $LASTEXITCODE | Should -Be -ExpectedValue 0

        # Get the updated version string
        $currentVersion = $(c:\Windows\wins.exe --version).Split(" ")[2]
        Log-Info "wins.exe version after suc execution: $currentVersion"
        $initialVersion -ne $currentVersion | Should -BeTrue
        $currentVersion -eq $currentVersion | Should -BeTrue

        # Ensure that the old file doesn't exist
        Test-Path "c:/etc/rancher/wins/wins-$initialVersion.exe" | Should -BeFalse
        # Ensure that the updated file was moved
        Test-Path "c:/etc/rancher/wins/wins-$currentVersion.exe" | Should -BeFalse

        Log-Info "Testing updated binaries..."
        # Ensure that both paths were updated
        $windowsDirVersion = $(c:\Windows\wins.exe --version).Split(" ")[2]
        $usrBinVersion = $(c:\usr\local\bin\wins.exe --version).Split(" ")[2]

        # Ensure that the version matches what we expect
        $windowsDirVersion -eq $currentVersion | Should -BeTrue
        $usrBinVersion -eq $currentVersion | Should -BeTrue

        # Ensure that the old binary no longer exists
        Log-Info "Ensuring old binary has been removed"
        $oldBinaryPath = "c:/etc/rancher/wins/wins-$initialVersion.exe"
        Test-Path $oldBinaryPath | Should -BeFalse

        Log-Info "Succesfully tested binary upgrade"
    }
}