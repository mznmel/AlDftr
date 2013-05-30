from cx_Freeze import setup, Executable

excludes = []
includefiles = ['./templates/', './static/', '../data/']
packages = ['flask', 'jinja2']

build_exe_options = {'excludes':excludes,'packages':packages,'include_files':includefiles}

setup(  name = "AlDftr",
        version = "0.1",
        description = "Wiki-Based Knowledge Organizer",
        author = 'Mazen A. Melibari',
        author_email = 'mazen@mazen.ws',
        options = {"build_exe": build_exe_options},
        executables = [Executable("AlDftr.py")])
