import os
import argparse

def generate_random_bytes(file_path: str, num_bytes: int):
    open(file_path, "wb").write(os.urandom(num_bytes))

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Generate random bytes")
    parser.add_argument("-o", "--output", dest="file_path", required=True, help="Output file path")
    parser.add_argument("-n", "--num-bytes", dest="num_bytes", type=int, default=1024, help="Number of random bytes to generate")
    args = parser.parse_args()

    generate_random_bytes(args.file_path, args.num_bytes)
    print(f"{args.num_bytes} random bytes written to {args.file_path}")
