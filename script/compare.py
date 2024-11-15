import numpy as np

def compare(file1, file2):
    diff = np.unpackbits(np.bitwise_xor(*map(lambda path: np.frombuffer(open(path, "rb").read(), dtype=np.uint8), (file1, file2))))
    return diff.sum(), diff.size

if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser(description="Compare two binary files")
    parser.add_argument("file1", help="First file to compare")
    parser.add_argument("file2", help="Second file to compare")
    args = parser.parse_args()

    count, total = compare(args.file1, args.file2)
    print(f"Files differ in {count} out of {total} bits")
