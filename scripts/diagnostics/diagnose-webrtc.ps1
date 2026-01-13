#!/usr/bin/env pwsh
# WebRTC Diagnostic Script

Write-Host "=== WebRTC Connection Diagnostics ===" -ForegroundColor Cyan
Write-Host ""

# 1. Check FreeSWITCH external_rtp_ip
Write-Host "1. FreeSWITCH External RTP IP:" -ForegroundColor Yellow
docker-compose exec -T freeswitch fs_cli -x 'global_getvar external_rtp_ip'
Write-Host ""

# 2. Check FreeSWITCH is listening on Verto port
Write-Host "2. FreeSWITCH Verto WebSocket (port 8081):" -ForegroundColor Yellow
docker-compose exec -T freeswitch netstat -tuln | Select-String "8081"
Write-Host ""

# 3. Check RTP port range
Write-Host "3. FreeSWITCH RTP Ports (sample 16384-16390):" -ForegroundColor Yellow
docker-compose exec -T freeswitch netstat -uln | Select-String "1638[0-9]" | Select-Object -First 5
Write-Host ""

# 4. Check Docker port mappings
Write-Host "4. Docker Port Mappings:" -ForegroundColor Yellow
docker port chatapp-freeswitch
Write-Host ""

# 5. Check ACL configuration
Write-Host "5. Verto ACL Configuration:" -ForegroundColor Yellow
docker-compose exec -T freeswitch fs_cli -x 'acl list' | Select-String -Pattern "localnet|rfc1918|any_v4" -Context 0,2
Write-Host ""

# 6. Test connectivity from host
Write-Host "6. Testing UDP connectivity to RTP ports:" -ForegroundColor Yellow
Write-Host "   Testing port 16384..." -NoNewline
try {
    $udpClient = New-Object System.Net.Sockets.UdpClient
    $udpClient.Connect("127.0.0.1", 16384)
    $udpClient.Close()
    Write-Host " OK" -ForegroundColor Green
} catch {
    Write-Host " FAIL: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# 7. Check Windows Firewall rules for RTP ports
Write-Host "7. Windows Firewall Rules for RTP:" -ForegroundColor Yellow
$firewallRules = Get-NetFirewallRule | Where-Object {
    $_.DisplayName -like "*Docker*" -or
    $_.DisplayName -like "*FreeSWITCH*" -or
    ($_.LocalPort -like "1638*" -or $_.LocalPort -like "8081")
}
if ($firewallRules) {
    $firewallRules | Select-Object DisplayName, Enabled, Direction, Action | Format-Table
} else {
    Write-Host "   No specific firewall rules found (may be allowed by Docker)" -ForegroundColor Gray
}
Write-Host ""

# 8. Summary
Write-Host "=== Summary ===" -ForegroundColor Cyan
Write-Host "For WebRTC to work with real IP (192.168.1.208):" -ForegroundColor White
Write-Host "  1. FreeSWITCH must have correct external_rtp_ip (should be 192.168.1.208)" -ForegroundColor White
Write-Host "  2. Ports 8081 (WS) and 16384-16484/udp (RTP) must be accessible" -ForegroundColor White
Write-Host "  3. Browser must be able to create UDP connections to these ports" -ForegroundColor White
Write-Host "  4. ACL must allow external IP in ICE candidates" -ForegroundColor White
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Green
Write-Host "  - If external_rtp_ip is wrong, set EXTERNAL_RTP_IP env var" -ForegroundColor Gray
Write-Host "  - If ports are blocked, check Windows Firewall" -ForegroundColor Gray
Write-Host "  - If using real IP from different machine, need STUN/TURN server" -ForegroundColor Gray
