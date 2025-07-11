function CleanupCertFile() {
    $path = "$PSScriptRoot\testcert.pem"
    Remove-Item -Path $path -Force
}

function SetupCertFiles() {
    param (
        [Parameter()]
        [int]
        $length
    )

    $certFilePath = "$PSScriptRoot\testcert.pem"

    Write-Host "Generating $length certificates"

    $root = New-SelfSignedCertificate -Subject "CN=wins-test-root-CA" -KeyUsage CertSign, CRLSign -KeyAlgorithm RSA -KeyLength 2048 -NotAfter (Get-Date).AddYears(1)
    if ($null -eq $root) {
        Write-Host "Failed to generate root certificate"
        exit 1
    }

    if ($length -eq 1) {
        $block = getCertString -cert $root
        Set-Content -Path $certFilePath -Value $block
        Write-Host "Generated single root CA certificate"
        return
    }

    [System.Security.Cryptography.X509Certificates.X509Certificate2[]]$certChain = @()
    $certChain += ($root)
    Write-Host "cert chain length" + $certChain.Length
    # Subtract 1 since we always include the root CA
    for ($i = 0; $i -lt $length-1; $i++) {
        if ($i -eq $length-2) {
            Write-Host "Generating final leaf certificate"
            $cert = newCert -parentCerts $certChain -isLeaf
            if ($null -ne $cert)
            {
                $certChain += ($cert)
            }
        } else
        {
            Write-Host "Generating intermediate certificate"
            $cert = newCert -parentCerts $certChain -isIntermediate
            if ($null -ne $cert)
            {
                $certChain += ($cert)
            }
        }
    }

    Write-Host "Generated $length certificates"
    foreach ($cert in $certChain) {
        $block = getCertString -cert $cert
        $finalCertFile = $finalCertFile + $block
    }

    # write final cert file to disk
    Set-Content -Path $certFilePath -Value $finalCertFile
    ls
    ls $PSScriptRoot

    $fingerPrints = @()

    Write-Host "Removing all generated certs from local cert store"
    foreach ($cert in $certChain) {
        $fingerPrints += @($cert.Fingerprint)
        Remove-Item -Path $cert.PSPath -DeleteKey
    }

    Write-Host "Deleted all certs from cert stores"
    return $fingerPrints
}

function getCertString() {
    param (
        [Parameter()]
        [System.Security.Cryptography.X509Certificates.X509Certificate2]
        $cert
    )

    if ($cert -eq $null) {
        Write-Host "BAD CERT BUDDY"
        exit 1
    }

    $base64Content = [System.Convert]::ToBase64String($cert.RawData, 'InsertLineBreaks')
    # The whitespace below is required
    # to ensure that newlines are properly added between
    # entries
    $newBlock = @"
-----BEGIN CERTIFICATE-----
$base64Content
-----END CERTIFICATE-----

"@

    return $newBlock
}

function newCert() {
    param (
        [Parameter()]
        [System.Security.Cryptography.X509Certificates.X509Certificate2[]]
        $parentCerts,

        [Parameter()]
        [Switch]
        $isLeaf,

        [Parameter()]
        [Switch]
        $isIntermediate
    )

    if ($isLeaf -and $isIntermediate) {
        Write-Host "a cert cannot be both a leaf and an intermediate"
        return
    }

    if ($isLeaf) {
        Write-Host "Creating a new leaf certificate"
        $cert = New-SelfSignedCertificate -Subject "CN=wins-test-leaf-cert" -DnsName "wins.com" -Signer $parentCerts[-1] -KeyAlgorithm RSA -KeyLength 2048 -NotAfter (Get-Date).AddYears(1)
        return $cert
    }

    if ($isIntermediate) {
        Write-Host "Creating a new intermediate certificate"
        $cert = New-SelfSignedCertificate -Subject "CN=wins-test-intermediate-cert" -KeyUsage CertSign, CRLSign -Signer $parentCerts[-1] -KeyAlgorithm RSA -KeyLength 2048 -NotAfter (Get-Date).AddYears(1)
        return $cert
    }

    return $null
}

Export-ModuleMember -Function SetupCertFiles
Export-ModuleMember -Function CleanupCertFile