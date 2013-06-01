"""
ALDftr: Wiki-Based Knowledge Organizer
"""
__author__ = "Mazen A. Melibari"
__email__ = "mazen@mazen.ws"
__license__ = "MPL"
__version__ = "0.1"

from flask import Flask, request, render_template, jsonify, redirect, url_for
import os
import sys
import re
import json
import codecs
import webbrowser

LOCAL_SERVER_PORT = 8001
ROOT_FOLDER = os.path.dirname(sys.argv[0])
DATA_FOLDER = unicode(os.path.join(ROOT_FOLDER, '../data')) # use unicode so that os.listdir() returns filenames in unicode!
METADATA_SEPARATOR = '#####-----|+|-|-|+|-----#####' # new lines added before and after the separator


#------------------------------------------------------------
# AlDftr's Main Class
#------------------------------------------------------------
class AlDftr:
    def __init__(self, data_folder, metadata_separator):
        self.data_folder = data_folder
        self.metadata_separator = metadata_separator

    def sanitize_page_name(self, page_name):
        """
        Returns a sanitized page name
        """
        # replace any nonalphanumeric(english and arabic) char or '_' by a dash
        #r'[^0-9a-zA-Z\u0600-\u06FF\_]+',
        regexp = re.compile(r'[^\w\d\:]+', re.UNICODE)
        return regexp.sub('-', page_name)

    def get_file_path(self, page_name):
        """
        Returns the full path of the page file
        """
        page_path_parts = page_name.split(':')
        page_path_parts[-1] = page_path_parts[-1] + '.txt'
        file_path = os.path.join(self.data_folder, *page_path_parts)
        return file_path

    def get_page(self, page_name):
        """
        Returns a tuple with  the content and metadata of the page
        """
        file_path = self.get_file_path(page_name)
        with codecs.open(file_path, 'r', 'utf-8') as file:
            file_content = file.read()

        (page_content, page_metadata) = self.parse_file_content(file_content)

        return (page_content, page_metadata)

    def save_page(self, page_name, page_content, page_metadata):
        file_path = self.get_file_path(page_name)

        #create namespace if not existed
        if not self.is_namespace_exists(file_path):
            os.makedirs(os.path.dirname(file_path))

        # append metadata to the end of the page
        file_content = page_content + '\n' + self.metadata_separator + '\n' + page_metadata
        with codecs.open(file_path, 'w', 'utf-8-sig') as file:
            file.write(file_content)

    def delete_page(self, page_name):
        file_path = self.get_file_path(page_name)
        os.remove(file_path)

    def is_page_exists(self, page_name):
        file_path = self.get_file_path(page_name)
        return os.path.exists(file_path)

    def is_namespace_exists(self, page_full_path):
        namespace_path = os.path.dirname(page_full_path)
        return os.path.exists(namespace_path)

    def get_all_pages(self):
        return os.listdir(self.data_folder)

    def parse_file_content(self, file_content):
        """
        Given a full page contents, this function separates the content and metadata and returns them
        """
        # prepare metadata sepration regular expression
        regexp = re.compile(r'\n' + re.escape(self.metadata_separator) + '.*', re.S)
        find_metadata = regexp.findall(file_content)
        if len(find_metadata):
            page_metadata = find_metadata[0]
            page_metadata = page_metadata.replace(self.metadata_separator, '')
            page_metadata = json.loads(page_metadata)
        else:
            page_metadata = {}

        # remove metadata from the content
        page_content = regexp.sub('', file_content)

        return (page_content, page_metadata)

    def dftr_format_to_html(self, text):
        """
        dftrFormat is a lightweight markup language that's inspired by Markdown, Wikiformat, and reStructuredText.
        It's designed specifically to be easily written in LTR and RTL languages.

        Current Specification (v0.001):
        \n                  : new line
        [[page-name]]       : local link to page-name
        **string**          : bold
        -----[-*]           : hr line
        #string             : h1
        ##string            : h2
        ###string           : h3
        ####string          : h4
        #####string         : h5
        ######string         : h6
        """

        text = text.replace('\r', '')

        local_link_regexp = re.compile('\[\[([^\]]*)\]\]')
        for page_name in local_link_regexp.findall(text):
            page_name = page_name.strip()
            page_link = '<a href="%s">%s</a>' % (url_for('view', page_name = page_name), page_name)
            text = text.replace('[[%s]]' % page_name, page_link)

        bold_regexp = re.compile(r'\*\*(.*?)\*\*')
        text = bold_regexp.sub(r'<b>\1</b>', text)

        hr_regexp = re.compile(r'^\-\-\-\-\-\-*$', re.M)
        text = hr_regexp.sub(r'<hr>', text)

        h6_regexp = re.compile(r'^\#\#\#\#\#\#(.*)$', re.M)
        text = h6_regexp.sub(r'<h6>\1</h6>', text)

        h5_regexp = re.compile(r'^\#\#\#\#\#(.*)$', re.M)
        text = h5_regexp.sub(r'<h5>\1</h5>', text)

        h4_regexp = re.compile(r'^\#\#\#\#(.*)$', re.M)
        text = h4_regexp.sub(r'<h4>\1</h4>', text)

        h3_regexp = re.compile(r'^\#\#\#(.*)$', re.M)
        text = h3_regexp.sub(r'<h3>\1</h3>', text)

        h2_regexp = re.compile(r'^\#\#(.*)$', re.M)
        text = h2_regexp.sub(r'<h2>\1</h2>', text)

        h1_regexp = re.compile(r'^\#(.*)$', re.M)
        text = h1_regexp.sub(r'<h1>\1</h1>', text)

        text = text.replace('\n', '<br>')

        return text

#------------------------------------------------------------
# Frontend
#------------------------------------------------------------

# Prepare global objects
aldftr = AlDftr(DATA_FOLDER, METADATA_SEPARATOR)
app = Flask(__name__)

# this function injects variables into all the templates.
# The 'endpoint' contains the name of the current action [index, view, or edit]
# we need it to change the look of the layout according to the current action
@app.context_processor
def inject():
    return dict(endpoint=request.endpoint, __version__=__version__)

# a wrapper around 'dftr_format_to_html' to convert from dftrFormat to html
@app.template_filter('dftr')
def dftr(s):
    return aldftr.dftr_format_to_html(s)


###----------------------------------------------------------
### Routes
###----------------------------------------------------------

@app.route('/')
def index():
    return redirect(url_for('view', page_name = 'main'))

@app.route('/all_pages')
def all_pages():
    pages = aldftr.get_all_pages()
    print(pages)
    return render_template('all_pages.html', pages = pages)

@app.route('/view/<page_name>')
def view(page_name):
    page_name = aldftr.sanitize_page_name(page_name)
    if aldftr.is_page_exists(page_name):
        (page_content, page_metadata) = aldftr.get_page(page_name)
        return render_template('view.html', page_name = page_name, page_content = page_content, page_metadata = page_metadata)
    else:
        return redirect(url_for('edit', page_name = page_name))

@app.route('/edit/<page_name>', methods=['POST', 'GET'])
def edit(page_name):
    if request.method == 'POST':
        page_name = aldftr.sanitize_page_name(request.form['page_name'])
        page_content = request.form['page_content']
        page_metadata = request.form['page_metadata']
        aldftr.save_page(page_name, page_content, page_metadata)

        return redirect(url_for('view', page_name = page_name))

    else:
        page_name = aldftr.sanitize_page_name(page_name)
        page_content = ''
        page_metadata = {}

        if aldftr.is_page_exists(page_name):
            (page_content, page_metadata) = aldftr.get_page(page_name)

        return render_template('edit.html', page_name = page_name, page_content = page_content, page_metadata = page_metadata)

@app.route('/delete/<page_name>')
def delete(page_name):
    page_name = aldftr.sanitize_page_name(page_name)
    aldftr.delete_page(page_name)
    return redirect(url_for('index'))


if __name__ == '__main__':
    print("#"*50)
    print("Aldftr v%s" % __version__)
    print("Please keep this window open as long as you want to access AlDftr")
    print("#"*50)

    app.debug = True
    webbrowser.open("http://localhost:8001")
    app.run(port=LOCAL_SERVER_PORT)
