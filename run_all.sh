surreal start --auth --user root --pass root --bind 127.0.0.1:8080 file:///db_data & 
surreal sql --conn http://127.0.0.1:8080 --user root --pass root --ns fyp --db violation_record --pretty &
# cd server && python discord_log_server.py &
cd client && python cam.py & 
cd server && python box.py 
# cd web && uvicorn web:app --reload --host localhost --port 8007 --log-level critical  
