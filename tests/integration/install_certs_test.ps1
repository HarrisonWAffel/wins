$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

$StartMockRancherHandler = {
    param(
        [Parameter()]
        [string]
        $certs
    )

    $http = New-Object System.Net.HttpListener
    $http.Prefixes.Add("http://localhost:8080/")
    $http.Start()

    while($http.IsListening) {
        $ctx = $http.GetContext()
        if ($ctx.Request.RawUrl -eq "/cacerts") {
            $buf = [System.Text.Encoding]::UTF8.GetBytes($certs)
            $ctx.response.ContentLength64 = $buf.Length
            $ctx.Response.OutputStream.Write($buf, 0, $buf.Length)
            $ctx.Response.OutputStream.Close()
        }
        # A dedicated kill endpoint works around a deadlock
        # that is encountered when Stop-Job is invoked at the same itme
        # that this function is waiting on GetContext()
        if ($ctx.Request.RawUrl -eq "/kill") {
            $ctx.Response.OutputStream.Close()
            exit 0
        }
    }
}

Describe "Install script certificate tests" {
    AfterEach {
        CleanupCertFile
        Log-Info "Running uninstall script"
        try {
            # note: since this script may not be run by an administrator, it's possible that it might fail
            # on trying to delete certain files with ACLs attached to them.
            # If you are running this locally, make sure you run with admin privileges.
            .\uninstall.ps1
        } catch {
            Log-Warn "You need to manually run uninstall.ps1, encountered error: $($_.Exception.Message)"
        }
    }

    BeforeEach {
        # Create a test specific copy of the install script
        # as the environment variables being set may differ between tests
        Copy-Item install.ps1 install-certs-test.ps1 -Force

        # note: Simply running the install script does not do anything. During normal provisioning,
        # Rancher will mutate the install script to both add environment variables, and to call
        # the primary function 'Invoke-WinsInstaller'. As this is an integration test, we need to manually
        # update the install script ourselves.
        Add-Content -Path ./installer-test.ps1 -Value '$env:CATTLE_REMOTE_ENABLED = "false"'
        Add-Content -Path ./installer-test.ps1 -Value '$env:CATTLE_LOCAL_ENABLED = "true"'
        Add-Content -Path ./install-certs-test.ps1 -Value "Invoke-WinsInstaller"
    }


    It "Imports a single cert properly" {
        Log-Info "TEST: [Imports a single cert properly]"
        $expectedCertificates = 1
        $certData = SetupCertFiles -length $expectedCertificates

        # Quick sanity check to ensure utility function properly removed
        # certificates from the built-in stores
        Log-Info "Ensuring that certs are not yet imported"
        foreach ($thumbPrint in $certData.ThumbPrints)
        {
            Log-Info "Ensuring cert with thumb print of $thumbPrint is not imported yet"
            $found = $certStore.Certificates.Find('FindByThumbprint', $thumbPrint, $false)
            $found.Count | Should -Be -ExpectedValue 0
            Log-Info "Cert with thumb print of $thumbPrint was not found in cert store"
        }

        $checkSum = $certData.Checksum
        PrependToFile -path "install-certs-test.ps1" -value "`$env:CATTLE_CA_CHECKSUM = `"$checkSum`""

        # Start a mock Rancher API to mimic the /cacerts endpoint
        Log-Info "Starting mock Rancher API"
        $job = Start-Job -ScriptBlock $StartMockRancherHandler -ArgumentList $certData.FinalCertBlocks
        Start-Sleep 1
        Get-Job
        if ($job.State -ne "Running") {
            $job | Receive-Job
            $job.State | Should -Be -ExpectedValue "Running"
        }

        Log-Info "Invoking install script function"
        .\install-certs-test.ps1
        $LASTEXITCODE | Should -Be -ExpectedValue 0

        # Stop the HTTP server
        Log-Info "Stopping mock server"
        curl.exe -sS http://localhost:8080/kill
        Remove-Job -Id $job.Id -Force

        Log-Info "Confirming that certs have been properly imported..."
        $certStore = [System.Security.Cryptography.X509Certificates.X509Store]::new([System.Security.Cryptography.X509Certificates.StoreName]::Root, "LocalMachine")
        $certStore.Open([System.Security.Cryptography.X509Certificates.OpenFlags]::MaxAllowed)

        foreach ($thumbPrint in $certData.ThumbPrints)
        {
            Log-Info "Checking cert with thumb print of $thumbPrint"
            $found = $certStore.Certificates.Find('FindByThumbprint', $thumbPrint, $false)
            $found.Count | Should -Be -ExpectedValue 1
            Log-Info "Found $thumbPrint"
        }

        $certStore.Close()
        Log-Info "Properly imported $expectedCertificates certificates"
    }

    It "Imports a chain properly" {
        Log-Info "TEST: [Imports a chain properly]"
        $expectedCertificates = 3
        $certData = SetupCertFiles -length $expectedCertificates

        # Quick sanity check to ensure utility function properly removed
        # certificates from the built-in stores
        Log-Info "Ensuring that certs are not yet imported"

        Log-Info $certData.ThumbPrints
        $certData.ThumbPrints.Length | Should -BeGreaterThan 0
        foreach ($thumbPrint in $certData.ThumbPrints)
        {
            Log-Info "Ensuring cert with thumb print of $thumbPrint is not imported yet"
            $found = $certStore.Certificates.Find('FindByThumbprint', $thumbPrint, $false)
            $found.Count | Should -Be -ExpectedValue 0
        }

        $checkSum = $certData.Checksum
        PrependToFile -path "install-certs-test.ps1" -value "`$env:CATTLE_CA_CHECKSUM = `"$checkSum`""
        Log-Info $(Get-Content install-certs-test.ps1)
        Log-Info "Starting mock Rancher API"
        $job = Start-Job -ScriptBlock $StartMockRancherHandler -ArgumentList $certData.FinalCertBlocks
        Start-Sleep 1
        Get-Job
        if ($job.State -ne "Running") {
            # display job output
            $job | Receive-Job
            $job.State | Should -Be -ExpectedValue "Running"
        }

        Log-Info "Invoking install script"

        .\install-certs-test.ps1
        $LASTEXITCODE | Should -Be -ExpectedValue 0

        # Stop the HTTP server
        Log-Info "Stopping mock server"
        curl.exe -sS http://localhost:8080/kill
        Remove-Job -Id $job.Id -Force

        Log-Info "Confirming that certs have been properly imported..."
        $certStore = [System.Security.Cryptography.X509Certificates.X509Store]::new([System.Security.Cryptography.X509Certificates.StoreName]::Root, "LocalMachine")
        $certStore.Open([System.Security.Cryptography.X509Certificates.OpenFlags]::MaxAllowed)

        $certData.ThumbPrints.Length | Should -BeGreaterThan 0
        foreach ($thumbPrint in $certData.ThumbPrints)
        {
            Log-Info "Checking cert with thumb print of $thumbPrint"
            $found = $certStore.Certificates.Find('FindByThumbprint', $thumbPrint, $false)
            $found.Count | Should -Be -ExpectedValue 1
            Log-Info "Found $thumbPrint"
        }

        $certStore.Close()
        Log-Info "Properly imported $expectedCertificates certificates"
    }
}
