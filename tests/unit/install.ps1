$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

Describe "Install script unit tests" {
    AfterEach {
        CleanupCertFile
    }

    It "Imports a single cert properly" {
        # We're using dot sourcing here as a hacky way to
        # get access to individual functions without
        # adding explicit export module statements in install.ps1
        . D:\a\wins\wins\install.ps1

        $expectedCertificates = 1
        Write-Host "Generating $expectedCertificates test certificates"
        $CertFingerPrints = SetupCertFiles -length $expectedCertificates

        $env:RANCHER_CERT = "$PSScriptRoot\testcert.pem"
        Write-Host "Invoking install script function"
        Import-Certificates

        Write-Host "Confirming that certs have been properly imported..."
        $certStore = [System.Security.Cryptography.X509Certificates.X509Store]::new([System.Security.Cryptography.X509Certificates.StoreName]::Root, "LocalMachine")
        $certStore.Open([System.Security.Cryptography.X509Certificates.OpenFlags]::MaxAllowed)

        foreach ($thumbPrint in $CertFingerPrints)
        {
            Write-Host "Checking cert with thumb print of $thumbPrint"
            $found = $certStore.Certificates.Find('FindByThumbprint', $thumbPrint, $false)
            $found.Count | Should -Be -ExpectedValue 1
            Write-Host "Found $thumbPrint"
        }

        $certStore.Close()
        Write-Host "Properly imported $expectedCertificates certificates"
    }

    It "Imports a chain properly" {
        # We're using dot sourcing here as a hacky way to
        # get access to individual functions without
        # adding explicit export module statements in install.ps1
        . D:\a\wins\wins\install.ps1

        Write-Host "Generating $expectedCertificates test certificates"
        $expectedCertificates = 3
        $CertFingerPrints = SetupCertFiles -length $expectedCertificates
        $env:RANCHER_CERT = "$(pwd)/testcert.pem"

        Write-Host "Invoking install script function"
        Import-Certificates

        Write-Host "Confirming that certs have been properly imported..."
        $certStore = [System.Security.Cryptography.X509Certificates.X509Store]::new([System.Security.Cryptography.X509Certificates.StoreName]::Root, "LocalMachine")
        $certStore.Open([System.Security.Cryptography.X509Certificates.OpenFlags]::MaxAllowed)

        foreach ($thumbPrint in $CertFingerPrints)
        {
            Write-Host "Checking cert with thumb print of $thumbPrint"
            $found = $certStore.Certificates.Find('FindByThumbprint', $thumbPrint, $false)
            $found.Count | Should -Be -ExpectedValue 1
            Write-Host "Found $thumbPrint"

        }

        $certStore.Close()
        Write-Host "Properly imported $expectedCertificates certificates"
    }
}
