# Home Assignment Step by Step How to Run the Code

## **1. Start the Server**

1. Open terminal (Git Bash or similar)
2. Navigate to the server folder:
   ```bash
   cd server
   ```
3. Run the server:
   ```bash
   go run main.go
   ```
4. You might get firewall request, just allow any.

## **2. Start the Client**

1. Open another terminal (I am using git bash)

You can either set environment variables first or use command below.

2. Navigate to the client folder and start the client:
   ```bash
   cd client
   ```
   
3. Run the client:
   ```bash
   CLIENT_ID=client1 SERVER_URL=http://localhost:8080 go run main.go
   ```
   Change the CLIENT_ID onto whatever client you want to activate and it will register/reactivate itself in ./server/clients.json
4. It will trigger the server to create or update clients.json in /server and poll the server every 10 seconds to make sure its active

## **3. Trigger a Download**
make an API call or curl from the GIT bash terminal.

   ```bash
   curl "http://localhost:8080/trigger?client_id=client1"
   ```

The client will upload its file to the server after 10 seconds (on poll).
After triggered, check for the server whether file is downloaded successfully or not

Change the client_id onto whichever client is active at the moment e.g. client2, client3

## **4. Check Uploaded File**
Uploaded files are saved in the server directory:
   ```bash
   ./downloads/client1_file.txt
   ```
file downloaded will vary based on which client server is triggering from, format is CLIENTNAME_file.txt

## **5. Notes**
**- Make sure file_to_download.txt exists in the main folder**

**- The current method is using polling instead of WebSocket due to time constraint, production is more proper using WebSocket**

**- The client automatically polls the server every 10 seconds, you can change the variable in client/main.go Line 35**

**- The server automatically refresh clients.json every 1 minute to check whether client is still active, you can change the variable in server/main.go Line 222**

**- Multiple clients can run simultaneously using different CLIENT_IDs, can just repeat number 2 and change the CLIENT_ID**












