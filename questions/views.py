from django.shortcuts import render
from django.views.decorators.csrf import csrf_exempt
from pypdf import PdfReader
# Create your views here.

@csrf_exempt
def index(request):
    if request.method == 'POST':
        file = request.FILES['doc']
        reader = PdfReader(file)
        text = "" 
        for page in reader.pages:
            text += page.extract_text()
        return render(request, "display.html", {"text": text})

    return render(request, 'index.html')

