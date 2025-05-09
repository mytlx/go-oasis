go build -o server.exe
Start-Process -NoNewWindow -FilePath "./server.exe" -ArgumentList "-port=8001"
Start-Process -NoNewWindow -FilePath "./server.exe" -ArgumentList "-port=8002"
Start-Process -NoNewWindow -FilePath "./server.exe" -ArgumentList "-port=8003 -api=1"

Start-Sleep -Seconds 2

Write-Host ">>> start test"
Start-Process curl "http://localhost:9999/api?key=Tom"
Start-Process curl "http://localhost:9999/api?key=Tom"
Start-Process curl "http://localhost:9999/api?key=Tom"

Start-Sleep -Seconds 5

Get-Process server | Stop-Process -Force
