# 🏗️ Smart Construction Site – AI-Powered Safety Monitoring System

This is a final-year project developed by a group of Higher Diploma in Software Engineering students at the Hong Kong Institute of Vocational Education (Tsing Yi). The goal of this project is to **enhance construction site safety** using real-time **AI-based detection and alerting systems**.

## 📌 Project Overview

The Smart Construction Site system leverages **computer vision and distributed computing** to monitor construction site conditions. It detects **safety violations**, such as missing helmets or vests, and also identifies **fires**, issuing **real-time alerts** via Discord to safety officers.


## 🧠 Technologies Used

| Component        | Technology             |
|------------------|------------------------|
| Object Detection | YOLOv8 (PyTorch)       |
| Image Processing | OpenCV                 |
| Backend API      | Golang Fiber           |
| Data Streaming   | gRPC                   |
| Communication    | Discord API            |
| Database         | SurrealDB              |
| Frontend         | HTML/CSS + JS (UI only)|
| Architecture     | Distributed system     |

## 🛠️ Features

- **Multiple Safety Inspections**  
  Detects helmets, masks, safety vests, people, machinery, and vehicles.

- **Fire Detection**  
  Real-time flame detection using a second YOLOv8 model.

- **Alerting System**  
  - Alerts for continuous safety violations.
  - Emergency broadcast in case of fire.
  - Periodic non-compliance reminders.

- **User Management**  
  Create, update, delete users with different roles (e.g. site manager, safety officer).

- **Data Analysis**  
  Safety compliance percentages, historical trends, and report generation.

- **Real-Time Monitoring**  
  24/7 surveillance using IP cams with instant browser updates.

## 🗂️ System Architecture

This system is designed using a **three-tier distributed architecture**:

1. **Presentation Layer:**  
   - Web UI using HTML, served by Golang Fiber
   - Displays detection results in real time

2. **Application Logic Layer:**  
   - Inference server using YOLOv8 (PyTorch) + gRPC
   - Cam server captures images and sends via gRPC in Base64
   - Golang web server handles client communication

3. **Data Layer:**  
   - SurrealDB stores metadata and violation logs
   - Discord API used for real-time alert distribution

## 📊 Dataset

- Object Detection: Roboflow dataset for construction safety (~2,800+ images)  
- Fire Detection: Pre-trained flame detection dataset (YOLO-compatible)  
- Helmet Color Classification (for role detection): Optional extension dataset

## 📁 How to Run (Simplified Overview)

1. **Cam Server** captures image → encodes in Base64 → sends via gRPC
2. **Inference Server** receives and decodes → runs YOLO model → sends results
3. **Web Server** listens via gRPC → updates frontend UI with detection results
4. **SurrealDB** logs violation info
5. **Discord API** pushes alerts

> ⚠️ Full installation and deployment steps are available in the `/docs` folder.

## 📸 UI Screenshots

- Login Page  
- Dashboard (Live Detection Feed)  
- Surveillance View  
- Log History View  





