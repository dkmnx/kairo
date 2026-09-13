#!/usr/bin/env pwsh
# PowerShell Escaping Integration Test Script
# Verifies that the Go EscapePowerShellArg contract — single-quote wrapping
# with ' doubling and NO other transformation — produces values that arrive
# intact and cannot break out of the quoting.

$ErrorActionPreference = "Stop"

Write-Host "=== PowerShell Escaping Integration Tests ===" -ForegroundColor Cyan
Write-Host ""

# Test 1: Basic string preservation
$test1 = 'hello'
if ($test1 -ne 'hello') { Write-Error 'Test 1 failed'; exit 1 }
Write-Host 'PASS: Basic string' -ForegroundColor Green

# Test 2: Single quote doubling
$test2 = 'can''t'
if ($test2 -ne "can't") { Write-Error 'Test 2 failed'; exit 1 }
Write-Host 'PASS: Single quote doubling' -ForegroundColor Green

# Test 3: Dollar sign is literal (NOT backtick-escaped)
$test3 = '$HOME'
if ($test3 -ne '$HOME') { Write-Error "Test 3 failed: got '$test3'"; exit 1 }
Write-Host 'PASS: Dollar sign literal' -ForegroundColor Green

# Test 4: Backtick is literal inside single quotes
$test4 = 'foo`bar'
if ($test4 -ne 'foo`bar') { Write-Error 'Test 4 failed'; exit 1 }
Write-Host 'PASS: Backtick literal' -ForegroundColor Green

# Test 5: Double quotes are literal (no backslash escaping)
$test5 = 'say "hi"'
if ($test5 -ne 'say "hi"') { Write-Error 'Test 5 failed'; exit 1 }
Write-Host 'PASS: Double quote literal' -ForegroundColor Green

# Test 6: Semicolon does not execute
$test6 = 'test; rm -rf /'
if ($test6 -ne 'test; rm -rf /') { Write-Error 'Test 6 failed'; exit 1 }
Write-Host 'PASS: Semicolon literal (not executed)' -ForegroundColor Green

# Test 7: Pipe does not execute
$test7 = 'test | calc'
if ($test7 -ne 'test | calc') { Write-Error 'Test 7 failed'; exit 1 }
Write-Host 'PASS: Pipe literal (not executed)' -ForegroundColor Green

# Test 8: Command substitution is literal
$test8 = '$(whoami)'
if ($test8 -ne '$(whoami)') { Write-Error "Test 8 failed: got '$test8'"; exit 1 }
Write-Host 'PASS: Command substitution literal' -ForegroundColor Green

# Test 9: Newline survives as a real newline (multi-line single-quoted string)
$test9 = 'line1
line2'
if ($test9 -ne "line1`nline2") { Write-Error 'Test 9 failed'; exit 1 }
Write-Host 'PASS: Newline preserved' -ForegroundColor Green

# Test 10: Injection payload cannot break out of single quotes
$test10 = '''; Start-Process calc; '''
if ($test10 -ne "'; Start-Process calc; '") { Write-Error 'Test 10 failed'; exit 1 }
Write-Host 'PASS: Quote breakout prevented' -ForegroundColor Green

# Test 11: Unicode emoji
$test11 = '🚀🎉'
if ($test11 -ne '🚀🎉') { Write-Error 'Test 11 failed'; exit 1 }
Write-Host 'PASS: Unicode emoji' -ForegroundColor Green

# Test 12: Unicode Chinese
$test12 = '你好世界'
if ($test12 -ne '你好世界') { Write-Error 'Test 12 failed'; exit 1 }
Write-Host 'PASS: Unicode Chinese' -ForegroundColor Green

Write-Host ""
Write-Host "=== All PowerShell Integration Tests Passed ===" -ForegroundColor Green
