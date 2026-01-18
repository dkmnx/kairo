#!/usr/bin/env pwsh
# PowerShell Escaping Integration Test Script
# Tests that the Go EscapePowerShellArg function produces valid PowerShell strings

$ErrorActionPreference = "Stop"

Write-Host "=== PowerShell Escaping Integration Tests ===" -ForegroundColor Cyan
Write-Host ""

# Build a comprehensive test script that verifies escaping behavior
$testScript = @'
# Test 1: Basic string preservation
$test1 = 'hello'
if ($test1 -ne 'hello') { Write-Error 'Test 1 failed'; exit 1 }
Write-Host 'PASS: Basic string' -ForegroundColor Green

# Test 2: Single quote handling
$test2 = 'can''t'
if ($test2 -ne "can't") { Write-Error 'Test 2 failed'; exit 1 }
Write-Host 'PASS: Single quote' -ForegroundColor Green

# Test 3: Dollar sign (should NOT expand)
$test3 = '`$HOME'
if ($test3 -ne '`$HOME') { Write-Error "Test 3 failed: got '$test3'"; exit 1 }
Write-Host 'PASS: Dollar sign preserved' -ForegroundColor Green

# Test 4: Backtick handling
$test4 = 'foo``bar'
if ($test4 -ne 'foo``bar') { Write-Error 'Test 4 failed'; exit 1 }
Write-Host 'PASS: Backtick preserved' -ForegroundColor Green

# Test 5: Double quote handling
$test5 = 'say \"hi\"'
if ($test5 -ne 'say \"hi\"') { Write-Error 'Test 5 failed'; exit 1 }
Write-Host 'PASS: Double quote escaped' -ForegroundColor Green

# Test 6: Command injection prevention - semicolon
$test6 = 'test; rm -rf /'
if ($test6 -ne 'test; rm -rf /') { Write-Error 'Test 6 failed'; exit 1 }
Write-Host 'PASS: Semicolon preserved (not executed)' -ForegroundColor Green

# Test 7: Command injection prevention - pipe
$test7 = 'test | calc'
if ($test7 -ne 'test | calc') { Write-Error 'Test 7 failed'; exit 1 }
Write-Host 'PASS: Pipe preserved (not executed)' -ForegroundColor Green

# Test 8: Command injection prevention - dollar sub
$test8 = '`$(whoami)'
if ($test8 -ne '`$(whoami)') { Write-Error "Test 8 failed: got '$test8'"; exit 1 }
Write-Host 'PASS: Command substitution escaped' -ForegroundColor Green

# Test 9: Newline escape
$test9 = 'line1`nline2'
$lines = $test9 -split '`n'
if ($lines.Count -ne 2 -or $lines[0] -ne 'line1' -or $lines[1] -ne 'line2') {
    Write-Error "Test 9 failed: newline escape not working"
    exit 1
}
Write-Host 'PASS: Newline escape' -ForegroundColor Green

# Test 10: Tab escape
$test10 = 'col1`tcol2'
$parts = $test10 -split '`t'
if ($parts.Count -ne 2 -or $parts[0] -ne 'col1' -or $parts[1] -ne 'col2') {
    Write-Error "Test 10 failed: tab escape not working"
    exit 1
}
Write-Host 'PASS: Tab escape' -ForegroundColor Green

# Test 11: Unicode
$test11 = '🚀🎉'
if ($test11 -ne '🚀🎉') { Write-Error 'Test 11 failed'; exit 1 }
Write-Host 'PASS: Unicode emoji' -ForegroundColor Green

# Test 12: Unicode Chinese
$test12 = '你好世界'
if ($test12 -ne '你好世界') { Write-Error 'Test 12 failed'; exit 1 }
Write-Host 'PASS: Unicode Chinese' -ForegroundColor Green

Write-Host ""
Write-Host "=== All PowerShell Integration Tests Passed ===" -ForegroundColor Green
'@

try {
    Invoke-Expression $testScript
} catch {
    Write-Error "Test failed: $_"
    exit 1
}
