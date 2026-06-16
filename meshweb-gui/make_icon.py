from PIL import Image
import sys

try:
    img = Image.open(r'E:\MeshWeb\assets\logo.png')
    icon_sizes = [(256, 256), (128, 128), (64, 64), (32, 32), (16, 16)]
    img.save(r'E:\MeshWeb\meshweb-gui\build\windows\icon.ico', format='ICO', sizes=icon_sizes)
    print('Icon generated successfully')
except Exception as e:
    print('Error:', e)
