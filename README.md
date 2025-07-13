Web Chat Application

Technology stack

Go,
PostgreSQL,
Redis,
Docker,
Docker-compose,
Prometheus,
Grafana

The project implements a robust registration and authentication system, with users divided into two distinct roles: Users and Administrators

User Features:
  Add other users as friends,
  Send and receive private messages,
  View posts from friends,
  Publish personal news

Admin Features:
  Block accounts via the admin panel,
  Gracefully shut down the server,
  All user features

WebSocket maintains persistent connections for live updates.

Grafana Dashboard:
   Current number of connected users,
   Message Delivery Latency,
   API Response Times
   
  
