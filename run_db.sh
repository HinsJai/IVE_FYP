surreal start --auth --user root --pass root --bind 127.0.0.1:8080 file:///db_data & 
surreal sql --conn http://127.0.0.1:8080 --user root --pass root --ns fyp --db violation_record --pretty
