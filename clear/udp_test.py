#!/usr/bin/env python3
import socket
import time

def test_udp_receiver():
    # Create UDP socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.bind(('0.0.0.0', 5000))
    sock.settimeout(5.0)  # 5 second timeout
    
    print("UDP listener started on port 5000...")
    print("Waiting for data from Android device...")
    
    packet_count = 0
    start_time = time.time()
    
    try:
        while True:
            try:
                data, addr = sock.recvfrom(65536)  # Large buffer
                packet_count += 1
                current_time = time.time()
                
                print(f"Packet #{packet_count} from {addr}: {len(data)} bytes")
                
                if packet_count == 1:
                    print(f"First packet received after {current_time - start_time:.2f} seconds")
                
                # Show first few bytes in hex
                hex_data = ' '.join(f'{b:02x}' for b in data[:16])
                print(f"  First 16 bytes: {hex_data}")
                
                if packet_count >= 10:
                    print("Received 10 packets successfully!")
                    break
                    
            except socket.timeout:
                elapsed = time.time() - start_time
                print(f"No data received in 5 seconds (total wait: {elapsed:.1f}s)")
                if elapsed > 30:  # Stop after 30 seconds total
                    print("Stopping after 30 seconds of no data")
                    break
                continue
                
    except KeyboardInterrupt:
        print("\nStopped by user")
    finally:
        sock.close()
        print("UDP socket closed")

if __name__ == "__main__":
    test_udp_receiver()