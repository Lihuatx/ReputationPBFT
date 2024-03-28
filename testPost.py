import requests

target_url = "http://localhost:1110/req"
post_data = b'{"clientID":"ahnhwi","operation":"GetMyName","timestamp":859381532}'
response = requests.post(target_url, data=post_data, headers={"Content-Type": "application/json"})

print(response)