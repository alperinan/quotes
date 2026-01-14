import os
import uuid
import shutil
from pathlib import Path

def generate_unique_string(length=16):
    """Generate a unique string of specified length"""
    return uuid.uuid4().hex[:length]

def rename_images_in_folder(folder_path="images"):
    """
    Rename all image files in the specified folder with unique 16-char strings
    """
    # Check if folder exists
    if not os.path.exists(folder_path):
        print(f"Error: Folder '{folder_path}' does not exist")
        return
    
    # Get all files in the folder
    files = [f for f in os.listdir(folder_path) if os.path.isfile(os.path.join(folder_path, f))]
    
    # Filter image files (common extensions)
    image_extensions = {'.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.tiff'}
    image_files = [f for f in files if Path(f).suffix.lower() in image_extensions]
    
    if not image_files:
        print(f"No image files found in '{folder_path}'")
        return
    
    print(f"Found {len(image_files)} image files in '{folder_path}'")
    print("=" * 80)
    
    renamed_count = 0
    errors = []
    used_names = set()
    
    for i, filename in enumerate(image_files, 1):
        old_path = os.path.join(folder_path, filename)
        
        # Get file extension
        extension = Path(filename).suffix.lower()
        if extension == '.jpeg':
            extension = '.jpg'  # Normalize jpeg to jpg
        
        # Generate unique name (ensure no collision)
        while True:
            unique_name = generate_unique_string(16)
            new_filename = f"{unique_name}{extension}"
            if new_filename not in used_names:
                used_names.add(new_filename)
                break
        
        new_path = os.path.join(folder_path, new_filename)
        
        try:
            # Rename the file
            os.rename(old_path, new_path)
            print(f"{i:3d}. {filename:40s} -> {new_filename}")
            renamed_count += 1
        except Exception as e:
            error_msg = f"Error renaming {filename}: {e}"
            print(f"{i:3d}. {error_msg}")
            errors.append(error_msg)
    
    # Summary
    print("=" * 80)
    print(f"\n✓ Successfully renamed {renamed_count}/{len(image_files)} files")
    
    if errors:
        print(f"\n⚠️  {len(errors)} errors occurred:")
        for error in errors:
            print(f"   - {error}")

def main():
    print("=" * 80)
    print("IMAGE FILE RENAMER")
    print("=" * 80)
    print("\nThis script will rename all image files in the 'images' folder")
    print("with unique 16-character strings.\n")
    
    # Ask for confirmation
    response = input("Do you want to proceed? (yes/no): ").strip().lower()
    
    if response == 'yes':
        rename_images_in_folder("images")
    else:
        print("\nOperation cancelled.")

if __name__ == "__main__":
    main()